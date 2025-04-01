# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package authz

import rego.v1

hasWriteAccess if {
    some role in input["realm_access/roles"] # iteration
    # We expect:
    # - with MT: [PROJECT_UUID]_en-agent-rw, [PROJECT_UUID]_en-ob or [PROJECT_UUID]_im-rw
    regex.match("^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)en-agent-rw$|^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_en-ob$|^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)im-rw$", role)
}

hasReadAccess if {
    some role in input["realm_access/roles"] # iteration
        # We expect:
        # - with MT: [PROJECT_UUID]_en-agent-rw, [PROJECT_UUID]_en-ob, [PROJECT_UUID]_im-r or [PROJECT_UUID]_im-rw
   regex.match("^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)en-agent-rw$|^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)en-ob$|^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)im-r$|^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)im-rw$", role)
}
