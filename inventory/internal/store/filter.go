// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// This file contains the code to convert AIP-160 filter expressions into ent
// SQL predicates, which are used in WHERE clauses to filter results.

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/structpb"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const (
	FunctionHas    = "has"
	binaryArgCount = 2
)

func noopSelector(*sql.Selector) {}

func noopSQLSelectorCreator(_ any) sqlPredicate {
	return noopSelector
}

var TypeNull = &expr.Type{TypeKind: &expr.Type_Null{Null: structpb.NullValue_NULL_VALUE}}

type Transpiler struct {
	ck *expr.CheckedExpr
	// kind contains the "root" resource kind we're transpiling for. Since we
	// expose the fields of a resource directly without a prefix, we need
	// to know which resource kind we're operating on for ident and select expressions.
	kind inv_v1.ResourceKind
}

func (t *Transpiler) Transpile() (func(*sql.Selector), error) {
	// No filter provided.
	if t.ck == nil {
		return noopSelector, nil
	}

	return t.transpileExpr(t.ck.GetExpr())
}

func (t *Transpiler) transpileExpr(e *expr.Expr) (func(*sql.Selector), error) {
	switch e.ExprKind.(type) {
	case *expr.Expr_CallExpr:
		return t.transpileCallExpr(e)
	case *expr.Expr_SelectExpr:
		return t.transpileSelectExpr(e)
	default:
		return nil, errors.Errorfc(codes.Unimplemented, "%v not implemented", e)
	}
}

// transpileRValueExpr takes an expr.Expr representing an R-Value and transpiles
// it to the contained value.
func (t *Transpiler) transpileRValueExpr(e *expr.Expr) (any, error) {
	switch e.GetExprKind().(type) {
	case *expr.Expr_ConstExpr:
		return t.transpileConstExpr(e)
	case *expr.Expr_IdentExpr:
		return t.transpileIdentExpr(e)
	default:
		return nil, errors.Errorfc(
			codes.InvalidArgument,
			"unexpected type of argument to RHS expression: %T",
			e.GetExprKind(),
		)
	}
}

func (t *Transpiler) transpileIdentExpr(e *expr.Expr) (any, error) {
	identExpr := e.GetIdentExpr()
	switch identExpr.GetName() {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil //nolint:nilnil // intentional
	default:
		return identExpr.GetName(), nil
	}
}

func (t *Transpiler) transpileConstExpr(e *expr.Expr) (any, error) {
	switch c := e.GetConstExpr().GetConstantKind().(type) {
	case *expr.Constant_Int64Value:
		return c.Int64Value, nil
	case *expr.Constant_Uint64Value:
		return c.Uint64Value, nil
	case *expr.Constant_StringValue:
		return c.StringValue, nil
	case *expr.Constant_BoolValue:
		return c.BoolValue, nil
	case *expr.Constant_NullValue:
		return nil, error(nil)
	default:
		return nil, errors.Errorfc(codes.InvalidArgument, "invalid type of constant `%v`", c)
	}
}

//nolint:cyclop // inherently complex
func (t *Transpiler) transpileCallExpr(e *expr.Expr) (func(*sql.Selector), error) {
	switch e.GetCallExpr().GetFunction() {
	case FunctionHas, filtering.FunctionHas:
		return t.transpileHasCallExpr(e)
	case filtering.FunctionEquals:
		return t.transpileComparisonCallExpr(e, sql.EQ)
	case filtering.FunctionNotEquals:
		return t.transpileComparisonCallExpr(e, sql.NEQ)
	case filtering.FunctionLessThan:
		return t.transpileComparisonCallExpr(e, sql.LT)
	case filtering.FunctionLessEquals:
		return t.transpileComparisonCallExpr(e, sql.LTE)
	case filtering.FunctionGreaterThan:
		return t.transpileComparisonCallExpr(e, sql.GT)
	case filtering.FunctionGreaterEquals:
		return t.transpileComparisonCallExpr(e, sql.GTE)
	case filtering.FunctionAnd:
		return t.transpileBinaryAndCallExpr(e)
	case filtering.FunctionOr:
		return t.transpileBinaryOrCallExpr(e)
	case filtering.FunctionNot:
		return t.transpileNotCallExpr(e)
	default:
		return nil, errors.Errorfc(codes.Unimplemented, "%v not implemented", e)
	}
}

func (t *Transpiler) transpileHasCallExpr(e *expr.Expr) (func(*sql.Selector), error) {
	zlog.Trace().Msgf("transpileHasCallExpr(%v)", e)
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 1 {
		return nil, errors.Errorfc(
			codes.InvalidArgument,
			"unexpected number of arguments to `%s` expression: %d",
			FunctionHas,
			len(callExpr.Args),
		)
	}

	switch callExpr.Args[0].ExprKind.(type) {
	case *expr.Expr_IdentExpr:
		return t.transpileHasCallIdentExpr(callExpr.Args[0])
	case *expr.Expr_SelectExpr:
		return t.transpileSelectExpr(callExpr.Args[0])
	default:
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown expression type %T", callExpr.Args[0].GetExprKind())
	}
}

// transpileHasCallIdentExpr is used to transpile expr.Expr_Ident inside a `has()` expression.
// This transpiles to a SQL `WHERE <edge> IS NOT NULL` predicate.
func (t *Transpiler) transpileHasCallIdentExpr(e *expr.Expr) (func(*sql.Selector), error) {
	zlog.Trace().Msgf("transpileHasCallIdentExpr(%v)", e)
	rt := resourceTranspilerRegistry.Get(t.kind)
	if rt == nil {
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown resource kind `%s`", t.kind)
	}

	edgeName := e.GetIdentExpr().GetName()
	predicate := rt.hasEdgeHandlerByEdgeName[edgeName]
	if predicate == nil {
		return nil, errors.Errorfc(codes.InvalidArgument, "invalid edge %v", edgeName)
	}
	return predicate, nil
}

func stringContainsOp(col string, v any) *sql.Predicate {
	zlog.Trace().Msgf("stringContainsOp(%v, %v)", col, v)
	s, ok := v.(string)
	if !ok {
		p := sql.P()
		p.AddError(errors.Errorfc(codes.Internal, "expected v to be of type string"))
		return p
	}
	// Don't do a wildcard match against empty strings.
	if s == "" {
		return sql.Or(sql.EQ(col, s), sql.IsNull(col)) // Treat NULL as "".
	}

	// Perform a fuzzy wildcard string search, by building a ILIKE expression.
	return sql.P().Append(func(b *sql.Builder) {
		if b.Dialect() != dialect.Postgres {
			b.AddError(errors.Errorfc(codes.Internal, "unsupported SQL dialect `%v`", b.Dialect()))
			return
		}
		// Escape special string matching chars and translate wildcards '*'.
		s = strings.ReplaceAll(s, "\\", "\\\\") // Backslashes must be escaped first, before introducing ours.
		s = strings.ReplaceAll(s, "%", "\\%")
		s = strings.ReplaceAll(s, "_", "\\_")
		s = strings.ReplaceAll(s, "*", "%")

		b.Ident(col).WriteString(" ILIKE ")
		b.Arg("%" + strings.ToLower(s) + "%")
	})
}

func fieldBoolComparisonOp(col string, v any) *sql.Predicate {
	zlog.Trace().Msgf("fieldBoolComparisonOp(%v, %v)", col, v)
	b, ok := v.(bool)
	if !ok {
		p := sql.P()
		p.AddError(errors.Errorfc(codes.Internal, "expected v to be of type bool"))
		return p
	}
	var p *sql.Predicate
	if b {
		// The And is for the NEQ comparison where we wrap this predicate with a Not.
		// `Not (True && NotNull)` is the same as  `Not True || Not NotNull`, simplified `False || Null`.
		p = sql.And(sql.IsTrue(col), sql.NotNull(col))
	} else {
		// The IsNull check is a workaround for bool fields in protobuf. "false" is always treated as
		// unset, hence we do not set boolean fields to false when building a mutation. Instead,
		// those remain "null".
		p = sql.Or(sql.IsFalse(col), sql.IsNull(col))
	}
	return p
}

//nolint:cyclop // handling special cases is complex.
func (t *Transpiler) transpileComparisonCallExpr(e *expr.Expr, op func(col string, v any) *sql.Predicate) (
	func(*sql.Selector), error,
) {
	zlog.Trace().Msgf("transpileComparisonCallExpr(%v)", e)
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != binaryArgCount {
		return nil, errors.Errorfc(
			codes.InvalidArgument,
			"unexpected number of arguments to `%s` expression: %d",
			callExpr.GetFunction(),
			len(callExpr.Args),
		)
	}
	rhsExpr, err := t.transpileRValueExpr(callExpr.Args[1])
	if err != nil {
		return nil, err
	}
	// On string equalities, ignore case and handle wildcards.
	if _, ok := rhsExpr.(string); ok && callExpr.GetFunction() == filtering.FunctionEquals {
		op = stringContainsOp
	}
	// On bool comparisons, threat null as false.
	if _, ok := rhsExpr.(bool); ok && callExpr.GetFunction() == filtering.FunctionEquals {
		op = fieldBoolComparisonOp
	}
	if _, ok := rhsExpr.(bool); ok && callExpr.GetFunction() == filtering.FunctionNotEquals {
		op = func(col string, v any) *sql.Predicate { return sql.Not(fieldBoolComparisonOp(col, v)) }
	}
	if rhsExpr == nil && callExpr.GetFunction() == filtering.FunctionEquals {
		op = func(col string, _ any) *sql.Predicate { return sql.IsNull(col) }
	}
	if rhsExpr == nil && callExpr.GetFunction() == filtering.FunctionNotEquals {
		op = func(col string, _ any) *sql.Predicate { return sql.NotNull(col) }
	}
	lhsExpr, err := t.transpileLValueExpr(callExpr.Args[0], op)
	if err != nil {
		return nil, err
	}

	return lhsExpr(rhsExpr), nil
}

func (t *Transpiler) transpileBinaryAndCallExpr(e *expr.Expr) (func(*sql.Selector), error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != binaryArgCount {
		return nil, errors.Errorfc(
			codes.InvalidArgument,
			"unexpected number of arguments to `%s` expression: %d",
			filtering.FunctionAnd,
			len(callExpr.Args),
		)
	}
	lhsExpr, err := t.transpileExpr(callExpr.Args[0])
	if err != nil {
		return nil, err
	}
	rhsExpr, err := t.transpileExpr(callExpr.Args[1])
	if err != nil {
		return nil, err
	}

	return func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		lhsExpr(s1)
		rhsExpr(s1)
		s.Where(s1.P())
	}, nil
}

func (t *Transpiler) transpileBinaryOrCallExpr(e *expr.Expr) (func(*sql.Selector), error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != binaryArgCount {
		return nil, errors.Errorfc(
			codes.InvalidArgument,
			"unexpected number of arguments to `%s` expression: %d",
			filtering.FunctionOr,
			len(callExpr.Args),
		)
	}
	lhsExpr, err := t.transpileExpr(callExpr.Args[0])
	if err != nil {
		return nil, err
	}
	rhsExpr, err := t.transpileExpr(callExpr.Args[1])
	if err != nil {
		return nil, err
	}

	return func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		s1.Or()
		lhsExpr(s1)
		s1.Or()
		rhsExpr(s1)
		s.Where(s1.P())
	}, nil
}

func (t *Transpiler) transpileNotCallExpr(e *expr.Expr) (func(*sql.Selector), error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 1 {
		return nil, errors.Errorfc(
			codes.InvalidArgument,
			"unexpected number of arguments to `%s` expression: %d",
			filtering.FunctionNot,
			len(callExpr.Args),
		)
	}
	rhsExpr, err := t.transpileExpr(callExpr.Args[0])
	if err != nil {
		return nil, err
	}
	return func(s *sql.Selector) {
		rhsExpr(s.Not())
	}, nil
}

func (t *Transpiler) transpileSelectExpr(e *expr.Expr) (func(*sql.Selector), error) {
	zlog.Trace().Msgf("transpileSelectExpr(%v)", e)
	noop := func(string, any) *sql.Predicate {
		return sql.P()
	}
	paths := flattenSelectExpr(e)
	pred, err := evaluate(t.kind, noop, paths)
	if err != nil {
		return nil, err
	}
	return pred(nil), nil
}

func (t *Transpiler) transpileLValueExpr(e *expr.Expr, op func(col string, v any) *sql.Predicate) (
	func(value any) sqlPredicate, error,
) {
	zlog.Trace().Msgf("transpileLValueExpr(%v)", e)
	var paths []string
	switch ex := e.ExprKind.(type) {
	case *expr.Expr_IdentExpr:
		paths = append(paths, ex.IdentExpr.GetName())
	case *expr.Expr_SelectExpr:
		paths = flattenSelectExpr(e)
	default:
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown expression type %T", e.GetExprKind())
	}

	return evaluate(t.kind, op, paths)
}

// flattenSelectExpr takes an expression and returns its nested fields as a flat list.
// Note that an expr.Expr_Select unravels from the head as the innermost element:
// For a given expression `host.site.region.resource_id`, the innermost expression
// will be `host.site`, not `region.resource_id`. This is cumbersome when building
// the ent SQL predicate, as the outermost predicate has to be `HasHostWith(...)`.
// By flattening the expression into a list, we can build the predicate in the
// right order. The exploded expression tree below illustrates the problem:
//
//	(op: (op: (op: host, fl: site), fl: region), fl: resource_id)  - host.site.region.resource_id
//	     (op: (op: host, fl: site), fl: region)                    - host.site.region
//	          (op: host, fl: site)                                 - host.site
//	           op: host                                            - host
//	        -> HasHostWith(...)
//	                     fl: site
//	        -> HasHostWith(hosts.HasSiteWith(...))
//	                               fl: region
//	        -> HasHostWith(hosts.HasSiteWith(sites.HasRegionWith(...)))
//	                               				 fl: resource_id
//	        -> HasHostWith(hosts.HasSiteWith(sites.HasRegionWith(regions.ResourceId(...))))
func flattenSelectExpr(e *expr.Expr) []string {
	selectExpr := e.GetSelectExpr()
	switch selectExpr.GetOperand().GetExprKind().(type) {
	case *expr.Expr_IdentExpr:
		return []string{selectExpr.GetOperand().GetIdentExpr().GetName(), selectExpr.GetField()}
	case *expr.Expr_SelectExpr:
		return append(flattenSelectExpr(selectExpr.GetOperand()), selectExpr.GetField())
	default:
		return nil
	}
}

// resourceFilter is a wrapper type to satisfy the filtering.Request interface.
type resourceFilter string

func (m *resourceFilter) GetFilter() string {
	return string(*m)
}

func isAllUpper(s string) bool {
	return strings.ToUpper(s) == s
}

func doNormalizeIdentExpr(e *expr.Expr) bool {
	identExpr := e.GetIdentExpr()
	if isAllUpper(identExpr.GetName()) {
		// Likely an enum constant.
		return false
	}
	if identExpr.GetName() == strcase.ToSnake(identExpr.GetName()) {
		// Already snake_case.
		return false
	}
	identExpr.Name = strcase.ToSnake(identExpr.GetName())
	// We have some fields that require special treatment after snake casing them.
	if identExpr.GetName() == "sha_256" {
		identExpr.Name = "sha256"
	}
	return true
}

func normalizeIdentNamesSnakeCaseIdentExpr(cursor *filtering.Cursor) {
	c, ok := proto.Clone(cursor.Expr()).(*expr.Expr)
	if !ok {
		zlog.Error().Msgf("expression clone failed")
		return
	}
	if doNormalizeIdentExpr(c) {
		cursor.Replace(c)
	}
	zlog.Trace().Msgf("replaced ident expr `%v` with `%v`", cursor.Expr(), c)
}

func doNormalizeSelectExpr(e *expr.Expr) bool {
	selectExpr := e.GetSelectExpr()
	if isAllUpper(selectExpr.GetField()) {
		// Likely an enum constant.
		return false
	}
	if selectExpr.GetField() == strcase.ToSnake(selectExpr.GetField()) {
		// Already snake_case.
		return false
	}
	selectExpr.Field = strcase.ToSnake(selectExpr.GetField())

	// Replace (nested) operand expression.
	switch selectExpr.GetOperand().GetExprKind().(type) {
	case *expr.Expr_IdentExpr:
		doNormalizeIdentExpr(selectExpr.GetOperand())
	case *expr.Expr_SelectExpr:
		doNormalizeSelectExpr(selectExpr.GetOperand())
	default:
		zlog.Trace().Msgf("unhandled nested select operand expression %v", selectExpr.GetOperand())
	}

	return true
}

func normalizeIdentNamesSnakeCaseSelectExpr(cursor *filtering.Cursor) {
	c, ok := proto.Clone(cursor.Expr()).(*expr.Expr)
	if !ok {
		zlog.Error().Msgf("expression clone failed")
		return
	}
	if doNormalizeSelectExpr(c) {
		cursor.Replace(c)
	}
	zlog.Trace().Msgf("replaced select expr `%v` with `%v`", cursor.Expr(), c)
}

func normalizeIdentNamesSnakeCase(cursor *filtering.Cursor) {
	switch cursor.Expr().GetExprKind().(type) {
	case *expr.Expr_IdentExpr:
		normalizeIdentNamesSnakeCaseIdentExpr(cursor)
	case *expr.Expr_SelectExpr:
		normalizeIdentNamesSnakeCaseSelectExpr(cursor)
	default:
	}
}

func getPredicate[T func(*sql.Selector)](resourceKind inv_v1.ResourceKind, filter string) (T, error) {
	decls, err := getDecls(resourceKind)
	if err != nil {
		return nil, err
	}

	return getPreds(resourceKind, decls, filter)
}

func getPreds(kind inv_v1.ResourceKind, decls *filtering.Declarations, filter string) (func(*sql.Selector), error) {
	mf := resourceFilter(filter)
	f, err := filtering.ParseFilter(&mf, decls)
	if err != nil {
		zlog.Error().Err(err).Msgf("parse of filter `%v` failed", filter)
		return nil, errors.Errorfc(codes.InvalidArgument, "%v", err)
	}
	// No filter provided.
	if f.CheckedExpr == nil {
		return noopSelector, nil
	}

	// Normalize identifiers to snake_case. While we do add declarations for both casings
	// for the initial parse stage, the later transpile stages cannot handle camelCase.
	f, err = filtering.ApplyMacros(f, decls, normalizeIdentNamesSnakeCase)
	if err != nil {
		return nil, err
	}
	zlog.Trace().Msgf("CheckedExpr after macros: %v", f.CheckedExpr)

	t := Transpiler{
		ck:   f.CheckedExpr,
		kind: kind,
	}
	pred, err := t.Transpile()
	if err != nil {
		zlog.Error().Err(err).Msg("transpile failed")
		return nil, err
	}

	return pred, nil
}

func newTypeProtoMsg(fullName protoreflect.FullName) *expr.Type {
	return &expr.Type{TypeKind: &expr.Type_MessageType{MessageType: string(fullName)}}
}

//nolint:cyclop // inherently complex
func getResourceDeclarations(desc protoreflect.MessageDescriptor, prefix string, limit int) (opts []filtering.DeclarationOption) {
	limit--
	if limit < 0 {
		return nil
	}

	var hasOverloads []*expr.Decl_FunctionDecl_Overload
	var eqOverloads []*expr.Decl_FunctionDecl_Overload
	var neqOverloads []*expr.Decl_FunctionDecl_Overload

	fds := desc.Fields()
	for i := 0; i < fds.Len(); i++ {
		d := fds.Get(i)
		switch d.Kind() {
		case protoreflect.EnumKind:
			if enumType, err := protoregistry.GlobalTypes.FindEnumByName(d.Enum().FullName()); err == nil {
				opts = append(opts, filtering.DeclareEnumIdent(prefix+string(d.Name()), enumType))
				if d.HasJSONName() && d.JSONName() != string(d.Name()) {
					opts = append(opts, filtering.DeclareEnumIdent(prefix+d.JSONName(), enumType))
				}
				eqOverloads = append(eqOverloads, filtering.NewFunctionOverload(
					filtering.FunctionEquals+"_"+string(d.FullName()),
					filtering.TypeBool, filtering.TypeEnum(enumType), TypeNull))
				neqOverloads = append(neqOverloads, filtering.NewFunctionOverload(
					filtering.FunctionNotEquals+"_"+string(d.FullName()),
					filtering.TypeBool, filtering.TypeEnum(enumType), TypeNull))
			} else {
				zlog.Error().Err(err).Msgf("could not find enum %v", d.Enum().FullName())
			}
		case protoreflect.MessageKind:
			// add sub-message itself as typed proto message.
			opts = append(opts, filtering.DeclareIdent(prefix+string(d.Name()), newTypeProtoMsg(d.Message().FullName())))
			if d.HasJSONName() && d.JSONName() != string(d.Name()) {
				// add sub-message itself as typed proto message.
				opts = append(opts, filtering.DeclareIdent(prefix+d.JSONName(), newTypeProtoMsg(d.Message().FullName())))
			}
			opts = append(opts, getResourceDeclarations(d.Message(), prefix+string(d.Name()+"."), limit)...)
			if d.HasJSONName() && d.JSONName() != string(d.Name()) {
				opts = append(opts, getResourceDeclarations(d.Message(), prefix+d.JSONName()+".", limit)...)
			}
			// `has` function overload for this resource.
			hasOverloads = append(hasOverloads, filtering.NewFunctionOverload(FunctionHas+"_"+string(d.Message().FullName()),
				filtering.TypeBool, newTypeProtoMsg(d.Message().FullName())))
		case protoreflect.StringKind:
			opts = append(opts, filtering.DeclareIdent(prefix+string(d.Name()), filtering.TypeString))
			if d.HasJSONName() && d.JSONName() != string(d.Name()) {
				opts = append(opts, filtering.DeclareIdent(prefix+d.JSONName(), filtering.TypeString))
			}
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Int32Kind, protoreflect.Int64Kind:
			opts = append(opts, filtering.DeclareIdent(prefix+string(d.Name()), filtering.TypeInt))
			if d.HasJSONName() && d.JSONName() != string(d.Name()) {
				opts = append(opts, filtering.DeclareIdent(prefix+d.JSONName(), filtering.TypeInt))
			}
		case protoreflect.BoolKind:
			opts = append(opts, filtering.DeclareIdent(prefix+string(d.Name()), filtering.TypeBool))
			if d.HasJSONName() && d.JSONName() != string(d.Name()) {
				opts = append(opts, filtering.DeclareIdent(prefix+d.JSONName(), filtering.TypeBool))
			}
		default:
			zlog.Trace().Msgf("skipped %s field %s, kind %v", desc.Name(), d.Name(), d.Kind())
			continue
		}
		zlog.Trace().Msgf("added %s field %v (%v), kind %v", string(desc.Name()), prefix+string(d.Name()),
			prefix+d.JSONName(), d.Kind())
	}

	opts = append(opts,
		// Declare `has(<edge>)` function with all collected overloads.
		filtering.DeclareFunction(FunctionHas, hasOverloads...),
		// Declare "=" function for "<enum field> = null" overloads.
		filtering.DeclareFunction(filtering.FunctionEquals, eqOverloads...),
		// Declare "!=" function for "<enum field> != null" overloads.
		filtering.DeclareFunction(filtering.FunctionNotEquals, neqOverloads...),
	)

	return opts
}

func getDeclOptions(desc protoreflect.MessageDescriptor) (opts []filtering.DeclarationOption) {
	const declarationDepth = 5
	opts = append(opts,
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("true", filtering.TypeBool),
		filtering.DeclareIdent("false", filtering.TypeBool),
		filtering.DeclareIdent("null", TypeNull),
	)
	opts = append(opts, getResourceDeclarations(desc, "", declarationDepth)...)

	return opts
}

// _declsOnce guard the initialization of _globalDecls.
var _declsOnce sync.Once

// _globalDecls hold the declarations for all resource kinds. It is initialized once on the first call
// of getDecls.
var _globalDecls = make(map[inv_v1.ResourceKind]struct {
	decls *filtering.Declarations
	err   error
})

//nolint:errcheck // prime the decl cache so the first request does not time out.
var _, _ = getDecls(inv_v1.ResourceKind_RESOURCE_KIND_UNSPECIFIED)

// getDecls returns the declarations for given resource kind.
func getDecls(kind inv_v1.ResourceKind) (*filtering.Declarations, error) {
	_declsOnce.Do(func() {
		for _, val := range inv_v1.ResourceKind_value {
			k := inv_v1.ResourceKind(val)
			if k == inv_v1.ResourceKind_RESOURCE_KIND_UNSPECIFIED {
				continue
			}
			d := _globalDecls[k]
			r, err := util.GetResourceFromKind(k)
			if err != nil {
				d.err = err
				_globalDecls[k] = d
				continue
			}

			inner, err := util.GetSetResource(r)
			if err != nil {
				d.err = err
				_globalDecls[k] = d
				continue
			}

			d.decls, d.err = filtering.NewDeclarations(getDeclOptions(inner.ProtoReflect().Descriptor())...)
			_globalDecls[k] = d
		}
	})

	d, found := _globalDecls[kind]
	if !found {
		return nil, errors.Errorfc(codes.NotFound, "no declarations for kind %v", kind)
	}

	return d.decls, d.err
}

type sqlPredicate = func(*sql.Selector)

// resourceTranspilerRegistry - all resource transpilers are automatically registered in this registry.
var resourceTranspilerRegistry *registry

type resourceTranspiler struct {
	// identifier of resource related with this transpiler
	id inv_v1.ResourceKind
	// resource's fields validator
	validateColumnFn validateColumnFn
	// set of resource edge handlers by edge name
	edgeHandlerByEdgeName    map[string]edgeHandler
	hasEdgeHandlerByEdgeName map[string]sqlPredicate
}

type validateColumnFn func(column string) bool

type deriveSelectorFn func(fn sqlPredicate) sqlPredicate

type edgeHandler struct {
	deriveSelectorFn deriveSelectorFn
	targetResourceID inv_v1.ResourceKind
}

func newResourceTranspiler(
	id inv_v1.ResourceKind,
	validateColumnFn validateColumnFn,
	edgeHandlers map[string]edgeHandler,
	hasEdgeHandlers map[string]sqlPredicate,
) *resourceTranspiler {
	return &resourceTranspiler{
		id:                       id,
		validateColumnFn:         validateColumnFn,
		edgeHandlerByEdgeName:    edgeHandlers,
		hasEdgeHandlerByEdgeName: hasEdgeHandlers,
	}
}

var registrarFunctionSignature = func(*registry) {}

func newRegistry() *registry {
	r := &registry{
		data: map[inv_v1.ResourceKind]*resourceTranspiler{},
	}
	callAllMethodsWithSignature(r, registrarFunctionSignature)
	return r
}

func callAllMethodsWithSignature(target, signature interface{}) {
	val := reflect.ValueOf(target)
	typ := val.Type()
	sig := reflect.TypeOf(signature)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		methodType := method.Type
		if methodType == sig {
			zlog.Info().Msgf("reflect: executing: %s", method.Name)
			method.Func.Call([]reflect.Value{val})
		}
	}
}

type registry struct {
	data map[inv_v1.ResourceKind]*resourceTranspiler
}

func (r *registry) Register(t *resourceTranspiler) {
	r.data[t.id] = t
}

func (r *registry) Get(kind inv_v1.ResourceKind) *resourceTranspiler {
	return r.data[kind]
}

type createSQLPredicateFn func(col string, value any) *sql.Predicate

type createPredicateFn func(value any) sqlPredicate

func evaluate(kind inv_v1.ResourceKind, op createSQLPredicateFn, paths []string) (createPredicateFn, error) {
	if len(paths) == 0 {
		return noopSQLSelectorCreator, nil
	}

	bt := resourceTranspilerRegistry.Get(kind)
	if bt == nil {
		return nil, fmt.Errorf("transpiler is missing for kind %v", kind)
	}

	head, tail := paths[0], paths[1:] // pop first element
	if bt.validateColumnFn(head) {
		return func(value any) sqlPredicate {
			return func(s *sql.Selector) {
				s.Where(op(s.C(head), value))
			}
		}, nil
	}

	context, ok := bt.edgeHandlerByEdgeName[head]
	if !ok {
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown edge %v", head)
	}

	evaluatedTail, err := evaluate(context.targetResourceID, op, tail)
	if err != nil {
		return nil, err
	}

	return func(value any) sqlPredicate {
		return context.deriveSelectorFn(evaluatedTail(value))
	}, nil
}
