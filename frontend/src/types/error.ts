export enum GeneralErrorCode {
  OK = 0,
  INTERNAL = 1,
  NOT_AUTHORIZED = 2,
  INVALID = 3,
  NOT_FOUND = 4,
  CONFLICT = 5,
  NOT_IMPLEMENTED = 6,
}

export enum DBErrorCode {
  CONNECTION_ERROR = 101,
  SYNTAX_ERROR = 102,
  EXECUTION_ERROR = 103,
}

export enum MigrationErrorCode {
  MIGRATION_SCHEMA_MISSING = 201,
  MIGRAITON_ALREADY_APPLIED = 202,
  MGIRATION_OUT_OF_ORDER = 203,
  MIGRATION_BASELINE_MISSING = 204,
}

export enum CompatibilityErrorCode {
  COMPATIBILITY_DROP_DATABASE = 10001,
  COMPATIBILITY_RENAME_TABLE = 10002,
  COMPATIBILITY_DROP_TABLE = 10003,
  COMPATIBILITY_RENAME_COLUMN = 10004,
  COMPATIBILITY_DROP_COLUMN = 10005,
  COMPATIBILITY_ADD_PRIMARY_KEY = 10006,
  COMPATIBILITY_ADD_UNIQUE_KEY = 10007,
  COMPATIBILITY_ADD_FOREIGN_KEY = 10008,
  COMPATIBILITY_ADD_CHECK = 10009,
  COMPATIBILITY_ALTER_CHECK = 10010,
  COMPATIBILITY_ALTER_COLUMN = 10011,
}

export type ErrorCode =
  | GeneralErrorCode
  | DBErrorCode
  | MigrationErrorCode
  | CompatibilityErrorCode;

export type ErrorTag = "General" | "Compatibility";

export type ErrorMeta = {
  code: ErrorCode;
  hash: string;
};
