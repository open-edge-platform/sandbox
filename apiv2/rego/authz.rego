# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package authz

import rego.v1

# Parses the input tenantid as a prefix in the write role.
# Iterates over the input roles, and for every expected write role,
# makes sure there is some role matching it.
# It supports only roles with tenantID prefix.
hasWriteAccess if {
    read_write_role := sprintf("%s_im-rw", [input.tenantid[0]])
    some role in input["realm_access/roles"] # iteration
    [read_write_role][_] == role
}

# Parses the input tenantid as a prefix in the read and read-write roles.
# Iterates over the input roles, and for every expected roles,
# makes sure there is some role matching it.
# It supports only roles with tenantID prefix.
hasReadAccess if {
    read_role := sprintf("%s_im-r", [input.tenantid[0]])
    read_write_role := sprintf("%s_im-rw", [input.tenantid[0]])
    some role in input["realm_access/roles"] # iteration
    [read_role, read_write_role][_] == role
}

