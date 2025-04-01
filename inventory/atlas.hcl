# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

lint {
    destructive {
        error = true
    }
    data_depend {
        error = true
    }
    incompatible {
        error = true
    }
    concurrent_index {
        error = true
    }
}

variable "PGUSER" {
    type    = string
    default = getenv("PGUSER")
}

variable "PGHOST" {
    type    = string
    default = getenv("PGHOST")
}

variable "PGDATABASE" {
    type    = string
    default = getenv("PGDATABASE")
}

variable "PGPORT" {
    type    = string
    default = getenv("PGPORT")
}

variable "PGPASSWORD" {
    type    = string
    default = getenv("PGPASSWORD")
}

variable "PGSSLMODE" {
    type    = string
    default = getenv("PGSSLMODE")
}

variable "MIGR_PATH" {
    type    = string
    default = getenv("MIGR_PATH")
}

env "local" {
    migration {
        dir = "file://${var.MIGR_PATH}"
    }
    dev = "postgres://${var.PGUSER}:${var.PGPASSWORD}@${var.PGHOST}:${var.PGPORT}/${var.PGDATABASE}?search_path=public&sslmode=${var.PGSSLMODE}"
    url = "postgres://${var.PGUSER}:${var.PGPASSWORD}@${var.PGHOST}:${var.PGPORT}/${var.PGDATABASE}?search_path=public&sslmode=${var.PGSSLMODE}"
}
