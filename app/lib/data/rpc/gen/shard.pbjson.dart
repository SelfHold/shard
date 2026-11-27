// This is a generated file - do not edit.
//
// Generated from shard.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use secretKindDescriptor instead')
const SecretKind$json = {
  '1': 'SecretKind',
  '2': [
    {'1': 'SECRET_KIND_UNSPECIFIED', '2': 0},
    {'1': 'SECRET_PASSWORD', '2': 1},
    {'1': 'SECRET_SSH', '2': 2},
    {'1': 'SECRET_WEB3', '2': 3},
    {'1': 'SECRET_TOTP', '2': 4},
    {'1': 'SECRET_SERVER', '2': 5},
  ],
};

/// Descriptor for `SecretKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List secretKindDescriptor = $convert.base64Decode(
    'CgpTZWNyZXRLaW5kEhsKF1NFQ1JFVF9LSU5EX1VOU1BFQ0lGSUVEEAASEwoPU0VDUkVUX1BBU1'
    'NXT1JEEAESDgoKU0VDUkVUX1NTSBACEg8KC1NFQ1JFVF9XRUIzEAMSDwoLU0VDUkVUX1RPVFAQ'
    'BBIRCg1TRUNSRVRfU0VSVkVSEAU=');

@$core.Deprecated('Use web3NetworkDescriptor instead')
const Web3Network$json = {
  '1': 'Web3Network',
  '2': [
    {'1': 'WEB3_NETWORK_UNSPECIFIED', '2': 0},
    {'1': 'WEB3_EVM', '2': 1},
    {'1': 'WEB3_SOLANA', '2': 2},
  ],
};

/// Descriptor for `Web3Network`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List web3NetworkDescriptor = $convert.base64Decode(
    'CgtXZWIzTmV0d29yaxIcChhXRUIzX05FVFdPUktfVU5TUEVDSUZJRUQQABIMCghXRUIzX0VWTR'
    'ABEg8KC1dFQjNfU09MQU5BEAI=');

@$core.Deprecated('Use serverAuthDescriptor instead')
const ServerAuth$json = {
  '1': 'ServerAuth',
  '2': [
    {'1': 'SERVER_AUTH_UNSPECIFIED', '2': 0},
    {'1': 'SERVER_AUTH_PASSWORD', '2': 1},
    {'1': 'SERVER_AUTH_SSH_KEY', '2': 2},
    {'1': 'SERVER_AUTH_AGENT', '2': 3},
  ],
};

/// Descriptor for `ServerAuth`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List serverAuthDescriptor = $convert.base64Decode(
    'CgpTZXJ2ZXJBdXRoEhsKF1NFUlZFUl9BVVRIX1VOU1BFQ0lGSUVEEAASGAoUU0VSVkVSX0FVVE'
    'hfUEFTU1dPUkQQARIXChNTRVJWRVJfQVVUSF9TU0hfS0VZEAISFQoRU0VSVkVSX0FVVEhfQUdF'
    'TlQQAw==');

@$core.Deprecated('Use secretImportStatusDescriptor instead')
const SecretImportStatus$json = {
  '1': 'SecretImportStatus',
  '2': [
    {'1': 'IMPORT_ADDED', '2': 0},
    {'1': 'IMPORT_DUPLICATE', '2': 1},
    {'1': 'IMPORT_FAILED', '2': 2},
  ],
};

/// Descriptor for `SecretImportStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List secretImportStatusDescriptor = $convert.base64Decode(
    'ChJTZWNyZXRJbXBvcnRTdGF0dXMSEAoMSU1QT1JUX0FEREVEEAASFAoQSU1QT1JUX0RVUExJQ0'
    'FURRABEhEKDUlNUE9SVF9GQUlMRUQQAg==');

@$core.Deprecated('Use emptyDescriptor instead')
const Empty$json = {
  '1': 'Empty',
};

/// Descriptor for `Empty`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List emptyDescriptor =
    $convert.base64Decode('CgVFbXB0eQ==');

@$core.Deprecated('Use stateDescriptor instead')
const State$json = {
  '1': 'State',
  '2': [
    {'1': 'vault_exists', '3': 1, '4': 1, '5': 8, '10': 'vaultExists'},
    {'1': 'unlocked', '3': 2, '4': 1, '5': 8, '10': 'unlocked'},
    {'1': 'vault_path', '3': 3, '4': 1, '5': 9, '10': 'vaultPath'},
    {'1': 'vault_id', '3': 4, '4': 1, '5': 9, '10': 'vaultId'},
    {'1': 'device_id', '3': 5, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'device_name', '3': 6, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'mount_point', '3': 7, '4': 1, '5': 9, '10': 'mountPoint'},
    {'1': 'platform', '3': 8, '4': 1, '5': 9, '10': 'platform'},
    {'1': 'ready', '3': 9, '4': 1, '5': 8, '10': 'ready'},
    {'1': 'problem', '3': 10, '4': 1, '5': 9, '10': 'problem'},
    {'1': 'mount_dir', '3': 11, '4': 1, '5': 9, '10': 'mountDir'},
    {'1': 'can_mount', '3': 12, '4': 1, '5': 8, '10': 'canMount'},
    {'1': 'core_version', '3': 13, '4': 1, '5': 9, '10': 'coreVersion'},
    {'1': 'core_build', '3': 14, '4': 1, '5': 9, '10': 'coreBuild'},
    {'1': 'opening', '3': 15, '4': 1, '5': 8, '10': 'opening'},
  ],
};

/// Descriptor for `State`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stateDescriptor = $convert.base64Decode(
    'CgVTdGF0ZRIhCgx2YXVsdF9leGlzdHMYASABKAhSC3ZhdWx0RXhpc3RzEhoKCHVubG9ja2VkGA'
    'IgASgIUgh1bmxvY2tlZBIdCgp2YXVsdF9wYXRoGAMgASgJUgl2YXVsdFBhdGgSGQoIdmF1bHRf'
    'aWQYBCABKAlSB3ZhdWx0SWQSGwoJZGV2aWNlX2lkGAUgASgJUghkZXZpY2VJZBIfCgtkZXZpY2'
    'VfbmFtZRgGIAEoCVIKZGV2aWNlTmFtZRIfCgttb3VudF9wb2ludBgHIAEoCVIKbW91bnRQb2lu'
    'dBIaCghwbGF0Zm9ybRgIIAEoCVIIcGxhdGZvcm0SFAoFcmVhZHkYCSABKAhSBXJlYWR5EhgKB3'
    'Byb2JsZW0YCiABKAlSB3Byb2JsZW0SGwoJbW91bnRfZGlyGAsgASgJUghtb3VudERpchIbCglj'
    'YW5fbW91bnQYDCABKAhSCGNhbk1vdW50EiEKDGNvcmVfdmVyc2lvbhgNIAEoCVILY29yZVZlcn'
    'Npb24SHQoKY29yZV9idWlsZBgOIAEoCVIJY29yZUJ1aWxkEhgKB29wZW5pbmcYDyABKAhSB29w'
    'ZW5pbmc=');

@$core.Deprecated('Use createVaultRequestDescriptor instead')
const CreateVaultRequest$json = {
  '1': 'CreateVaultRequest',
  '2': [
    {'1': 'password', '3': 1, '4': 1, '5': 9, '10': 'password'},
    {'1': 'data_shards', '3': 2, '4': 1, '5': 5, '10': 'dataShards'},
    {'1': 'parity_shards', '3': 3, '4': 1, '5': 5, '10': 'parityShards'},
    {'1': 'device_name', '3': 4, '4': 1, '5': 9, '10': 'deviceName'},
  ],
};

/// Descriptor for `CreateVaultRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createVaultRequestDescriptor = $convert.base64Decode(
    'ChJDcmVhdGVWYXVsdFJlcXVlc3QSGgoIcGFzc3dvcmQYASABKAlSCHBhc3N3b3JkEh8KC2RhdG'
    'Ffc2hhcmRzGAIgASgFUgpkYXRhU2hhcmRzEiMKDXBhcml0eV9zaGFyZHMYAyABKAVSDHBhcml0'
    'eVNoYXJkcxIfCgtkZXZpY2VfbmFtZRgEIAEoCVIKZGV2aWNlTmFtZQ==');

@$core.Deprecated('Use importVaultRequestDescriptor instead')
const ImportVaultRequest$json = {
  '1': 'ImportVaultRequest',
  '2': [
    {'1': 'vault', '3': 1, '4': 1, '5': 12, '10': 'vault'},
    {'1': 'password', '3': 2, '4': 1, '5': 9, '10': 'password'},
    {'1': 'device_name', '3': 3, '4': 1, '5': 9, '10': 'deviceName'},
  ],
};

/// Descriptor for `ImportVaultRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List importVaultRequestDescriptor = $convert.base64Decode(
    'ChJJbXBvcnRWYXVsdFJlcXVlc3QSFAoFdmF1bHQYASABKAxSBXZhdWx0EhoKCHBhc3N3b3JkGA'
    'IgASgJUghwYXNzd29yZBIfCgtkZXZpY2VfbmFtZRgDIAEoCVIKZGV2aWNlTmFtZQ==');

@$core.Deprecated('Use unlockRequestDescriptor instead')
const UnlockRequest$json = {
  '1': 'UnlockRequest',
  '2': [
    {'1': 'password', '3': 1, '4': 1, '5': 9, '10': 'password'},
    {'1': 'device_name', '3': 2, '4': 1, '5': 9, '10': 'deviceName'},
  ],
};

/// Descriptor for `UnlockRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List unlockRequestDescriptor = $convert.base64Decode(
    'Cg1VbmxvY2tSZXF1ZXN0EhoKCHBhc3N3b3JkGAEgASgJUghwYXNzd29yZBIfCgtkZXZpY2Vfbm'
    'FtZRgCIAEoCVIKZGV2aWNlTmFtZQ==');

@$core.Deprecated('Use changePasswordRequestDescriptor instead')
const ChangePasswordRequest$json = {
  '1': 'ChangePasswordRequest',
  '2': [
    {'1': 'old_password', '3': 1, '4': 1, '5': 9, '10': 'oldPassword'},
    {'1': 'new_password', '3': 2, '4': 1, '5': 9, '10': 'newPassword'},
  ],
};

/// Descriptor for `ChangePasswordRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changePasswordRequestDescriptor = $convert.base64Decode(
    'ChVDaGFuZ2VQYXNzd29yZFJlcXVlc3QSIQoMb2xkX3Bhc3N3b3JkGAEgASgJUgtvbGRQYXNzd2'
    '9yZBIhCgxuZXdfcGFzc3dvcmQYAiABKAlSC25ld1Bhc3N3b3Jk');

@$core.Deprecated('Use checkPasswordRequestDescriptor instead')
const CheckPasswordRequest$json = {
  '1': 'CheckPasswordRequest',
  '2': [
    {'1': 'password', '3': 1, '4': 1, '5': 9, '10': 'password'},
  ],
};

/// Descriptor for `CheckPasswordRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List checkPasswordRequestDescriptor =
    $convert.base64Decode(
        'ChRDaGVja1Bhc3N3b3JkUmVxdWVzdBIaCghwYXNzd29yZBgBIAEoCVIIcGFzc3dvcmQ=');

@$core.Deprecated('Use vaultFileDescriptor instead')
const VaultFile$json = {
  '1': 'VaultFile',
  '2': [
    {'1': 'vault', '3': 1, '4': 1, '5': 12, '10': 'vault'},
  ],
};

/// Descriptor for `VaultFile`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List vaultFileDescriptor =
    $convert.base64Decode('CglWYXVsdEZpbGUSFAoFdmF1bHQYASABKAxSBXZhdWx0');

@$core.Deprecated('Use backendOptionDescriptor instead')
const BackendOption$json = {
  '1': 'BackendOption',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'help', '3': 2, '4': 1, '5': 9, '10': 'help'},
    {'1': 'default', '3': 3, '4': 1, '5': 9, '10': 'default'},
    {'1': 'required', '3': 4, '4': 1, '5': 8, '10': 'required'},
    {'1': 'password', '3': 5, '4': 1, '5': 8, '10': 'password'},
    {'1': 'advanced', '3': 6, '4': 1, '5': 8, '10': 'advanced'},
    {'1': 'examples', '3': 7, '4': 3, '5': 9, '10': 'examples'},
  ],
};

/// Descriptor for `BackendOption`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backendOptionDescriptor = $convert.base64Decode(
    'Cg1CYWNrZW5kT3B0aW9uEhIKBG5hbWUYASABKAlSBG5hbWUSEgoEaGVscBgCIAEoCVIEaGVscB'
    'IYCgdkZWZhdWx0GAMgASgJUgdkZWZhdWx0EhoKCHJlcXVpcmVkGAQgASgIUghyZXF1aXJlZBIa'
    'CghwYXNzd29yZBgFIAEoCFIIcGFzc3dvcmQSGgoIYWR2YW5jZWQYBiABKAhSCGFkdmFuY2VkEh'
    'oKCGV4YW1wbGVzGAcgAygJUghleGFtcGxlcw==');

@$core.Deprecated('Use backendTypeDescriptor instead')
const BackendType$json = {
  '1': 'BackendType',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'description', '3': 2, '4': 1, '5': 9, '10': 'description'},
    {'1': 'oauth', '3': 3, '4': 1, '5': 8, '10': 'oauth'},
    {
      '1': 'options',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.BackendOption',
      '10': 'options'
    },
  ],
};

/// Descriptor for `BackendType`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backendTypeDescriptor = $convert.base64Decode(
    'CgtCYWNrZW5kVHlwZRISCgRuYW1lGAEgASgJUgRuYW1lEiAKC2Rlc2NyaXB0aW9uGAIgASgJUg'
    'tkZXNjcmlwdGlvbhIUCgVvYXV0aBgDIAEoCFIFb2F1dGgSOgoHb3B0aW9ucxgEIAMoCzIgLnNl'
    'bGZob2xkLnNoYXJkLnYxLkJhY2tlbmRPcHRpb25SB29wdGlvbnM=');

@$core.Deprecated('Use backendTypesReplyDescriptor instead')
const BackendTypesReply$json = {
  '1': 'BackendTypesReply',
  '2': [
    {
      '1': 'types',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.BackendType',
      '10': 'types'
    },
  ],
};

/// Descriptor for `BackendTypesReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backendTypesReplyDescriptor = $convert.base64Decode(
    'ChFCYWNrZW5kVHlwZXNSZXBseRI0CgV0eXBlcxgBIAMoCzIeLnNlbGZob2xkLnNoYXJkLnYxLk'
    'JhY2tlbmRUeXBlUgV0eXBlcw==');

@$core.Deprecated('Use addBackendRequestDescriptor instead')
const AddBackendRequest$json = {
  '1': 'AddBackendRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'type', '3': 2, '4': 1, '5': 9, '10': 'type'},
    {'1': 'domain', '3': 3, '4': 1, '5': 9, '10': 'domain'},
    {'1': 'root', '3': 4, '4': 1, '5': 9, '10': 'root'},
    {
      '1': 'params',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.AddBackendRequest.ParamsEntry',
      '10': 'params'
    },
    {'1': 'account', '3': 6, '4': 1, '5': 9, '10': 'account'},
  ],
  '3': [AddBackendRequest_ParamsEntry$json],
};

@$core.Deprecated('Use addBackendRequestDescriptor instead')
const AddBackendRequest_ParamsEntry$json = {
  '1': 'ParamsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `AddBackendRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List addBackendRequestDescriptor = $convert.base64Decode(
    'ChFBZGRCYWNrZW5kUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEhIKBHR5cGUYAiABKAlSBH'
    'R5cGUSFgoGZG9tYWluGAMgASgJUgZkb21haW4SEgoEcm9vdBgEIAEoCVIEcm9vdBJICgZwYXJh'
    'bXMYBSADKAsyMC5zZWxmaG9sZC5zaGFyZC52MS5BZGRCYWNrZW5kUmVxdWVzdC5QYXJhbXNFbn'
    'RyeVIGcGFyYW1zEhgKB2FjY291bnQYBiABKAlSB2FjY291bnQaOQoLUGFyYW1zRW50cnkSEAoD'
    'a2V5GAEgASgJUgNrZXkSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use idRequestDescriptor instead')
const IdRequest$json = {
  '1': 'IdRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
  ],
};

/// Descriptor for `IdRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List idRequestDescriptor =
    $convert.base64Decode('CglJZFJlcXVlc3QSDgoCaWQYASABKAlSAmlk');

@$core.Deprecated('Use removeBackendRequestDescriptor instead')
const RemoveBackendRequest$json = {
  '1': 'RemoveBackendRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'force', '3': 2, '4': 1, '5': 8, '10': 'force'},
  ],
};

/// Descriptor for `RemoveBackendRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeBackendRequestDescriptor = $convert.base64Decode(
    'ChRSZW1vdmVCYWNrZW5kUmVxdWVzdBIOCgJpZBgBIAEoCVICaWQSFAoFZm9yY2UYAiABKAhSBW'
    'ZvcmNl');

@$core.Deprecated('Use authorizeRequestDescriptor instead')
const AuthorizeRequest$json = {
  '1': 'AuthorizeRequest',
  '2': [
    {'1': 'type', '3': 1, '4': 1, '5': 9, '10': 'type'},
    {
      '1': 'params',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.AuthorizeRequest.ParamsEntry',
      '10': 'params'
    },
  ],
  '3': [AuthorizeRequest_ParamsEntry$json],
};

@$core.Deprecated('Use authorizeRequestDescriptor instead')
const AuthorizeRequest_ParamsEntry$json = {
  '1': 'ParamsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `AuthorizeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List authorizeRequestDescriptor = $convert.base64Decode(
    'ChBBdXRob3JpemVSZXF1ZXN0EhIKBHR5cGUYASABKAlSBHR5cGUSRwoGcGFyYW1zGAIgAygLMi'
    '8uc2VsZmhvbGQuc2hhcmQudjEuQXV0aG9yaXplUmVxdWVzdC5QYXJhbXNFbnRyeVIGcGFyYW1z'
    'GjkKC1BhcmFtc0VudHJ5EhAKA2tleRgBIAEoCVIDa2V5EhQKBXZhbHVlGAIgASgJUgV2YWx1ZT'
    'oCOAE=');

@$core.Deprecated('Use authorizeReplyDescriptor instead')
const AuthorizeReply$json = {
  '1': 'AuthorizeReply',
  '2': [
    {'1': 'url', '3': 1, '4': 1, '5': 9, '10': 'url'},
    {'1': 'token', '3': 2, '4': 1, '5': 9, '10': 'token'},
    {'1': 'account', '3': 3, '4': 1, '5': 9, '10': 'account'},
  ],
};

/// Descriptor for `AuthorizeReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List authorizeReplyDescriptor = $convert.base64Decode(
    'Cg5BdXRob3JpemVSZXBseRIQCgN1cmwYASABKAlSA3VybBIUCgV0b2tlbhgCIAEoCVIFdG9rZW'
    '4SGAoHYWNjb3VudBgDIAEoCVIHYWNjb3VudA==');

@$core.Deprecated('Use renameBackendRequestDescriptor instead')
const RenameBackendRequest$json = {
  '1': 'RenameBackendRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `RenameBackendRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List renameBackendRequestDescriptor = $convert.base64Decode(
    'ChRSZW5hbWVCYWNrZW5kUmVxdWVzdBIOCgJpZBgBIAEoCVICaWQSEgoEbmFtZRgCIAEoCVIEbm'
    'FtZQ==');

@$core.Deprecated('Use setBackendEnabledRequestDescriptor instead')
const SetBackendEnabledRequest$json = {
  '1': 'SetBackendEnabledRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'enabled', '3': 2, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `SetBackendEnabledRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setBackendEnabledRequestDescriptor =
    $convert.base64Decode(
        'ChhTZXRCYWNrZW5kRW5hYmxlZFJlcXVlc3QSDgoCaWQYASABKAlSAmlkEhgKB2VuYWJsZWQYAi'
        'ABKAhSB2VuYWJsZWQ=');

@$core.Deprecated('Use backendDescriptor instead')
const Backend$json = {
  '1': 'Backend',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'type', '3': 3, '4': 1, '5': 9, '10': 'type'},
    {'1': 'domain', '3': 4, '4': 1, '5': 9, '10': 'domain'},
    {'1': 'enabled', '3': 5, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'online', '3': 6, '4': 1, '5': 8, '10': 'online'},
    {'1': 'behind', '3': 7, '4': 1, '5': 8, '10': 'behind'},
    {'1': 'error', '3': 8, '4': 1, '5': 9, '10': 'error'},
    {'1': 'account', '3': 9, '4': 1, '5': 9, '10': 'account'},
    {'1': 'root', '3': 10, '4': 1, '5': 9, '10': 'root'},
    {'1': 'same_as', '3': 11, '4': 1, '5': 9, '10': 'sameAs'},
    {'1': 'preparing', '3': 12, '4': 1, '5': 8, '10': 'preparing'},
    {
      '1': 'total_bytes',
      '3': 13,
      '4': 1,
      '5': 3,
      '9': 0,
      '10': 'totalBytes',
      '17': true
    },
    {
      '1': 'free_bytes',
      '3': 14,
      '4': 1,
      '5': 3,
      '9': 1,
      '10': 'freeBytes',
      '17': true
    },
    {'1': 'stored_bytes', '3': 15, '4': 1, '5': 3, '10': 'storedBytes'},
    {'1': 'get_ms', '3': 16, '4': 1, '5': 5, '10': 'getMs'},
    {'1': 'put_ms', '3': 17, '4': 1, '5': 5, '10': 'putMs'},
    {'1': 'list_ms', '3': 18, '4': 1, '5': 5, '10': 'listMs'},
    {'1': 'oauth', '3': 19, '4': 1, '5': 8, '10': 'oauth'},
  ],
  '8': [
    {'1': '_total_bytes'},
    {'1': '_free_bytes'},
  ],
};

/// Descriptor for `Backend`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backendDescriptor = $convert.base64Decode(
    'CgdCYWNrZW5kEg4KAmlkGAEgASgJUgJpZBISCgRuYW1lGAIgASgJUgRuYW1lEhIKBHR5cGUYAy'
    'ABKAlSBHR5cGUSFgoGZG9tYWluGAQgASgJUgZkb21haW4SGAoHZW5hYmxlZBgFIAEoCFIHZW5h'
    'YmxlZBIWCgZvbmxpbmUYBiABKAhSBm9ubGluZRIWCgZiZWhpbmQYByABKAhSBmJlaGluZBIUCg'
    'VlcnJvchgIIAEoCVIFZXJyb3ISGAoHYWNjb3VudBgJIAEoCVIHYWNjb3VudBISCgRyb290GAog'
    'ASgJUgRyb290EhcKB3NhbWVfYXMYCyABKAlSBnNhbWVBcxIcCglwcmVwYXJpbmcYDCABKAhSCX'
    'ByZXBhcmluZxIkCgt0b3RhbF9ieXRlcxgNIAEoA0gAUgp0b3RhbEJ5dGVziAEBEiIKCmZyZWVf'
    'Ynl0ZXMYDiABKANIAVIJZnJlZUJ5dGVziAEBEiEKDHN0b3JlZF9ieXRlcxgPIAEoA1ILc3Rvcm'
    'VkQnl0ZXMSFQoGZ2V0X21zGBAgASgFUgVnZXRNcxIVCgZwdXRfbXMYESABKAVSBXB1dE1zEhcK'
    'B2xpc3RfbXMYEiABKAVSBmxpc3RNcxIUCgVvYXV0aBgTIAEoCFIFb2F1dGhCDgoMX3RvdGFsX2'
    'J5dGVzQg0KC19mcmVlX2J5dGVz');

@$core.Deprecated('Use deviceDescriptor instead')
const Device$json = {
  '1': 'Device',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'platform', '3': 3, '4': 1, '5': 9, '10': 'platform'},
    {'1': 'added_unix', '3': 4, '4': 1, '5': 3, '10': 'addedUnix'},
  ],
};

/// Descriptor for `Device`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceDescriptor = $convert.base64Decode(
    'CgZEZXZpY2USDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbWUSGgoIcGxhdGZvcm'
    '0YAyABKAlSCHBsYXRmb3JtEh0KCmFkZGVkX3VuaXgYBCABKANSCWFkZGVkVW5peA==');

@$core.Deprecated('Use statusReplyDescriptor instead')
const StatusReply$json = {
  '1': 'StatusReply',
  '2': [
    {'1': 'data_shards', '3': 1, '4': 1, '5': 5, '10': 'dataShards'},
    {'1': 'parity_shards', '3': 2, '4': 1, '5': 5, '10': 'parityShards'},
    {
      '1': 'backends',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.Backend',
      '10': 'backends'
    },
    {
      '1': 'devices',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.Device',
      '10': 'devices'
    },
    {'1': 'pending', '3': 5, '4': 1, '5': 5, '10': 'pending'},
    {'1': 'reload_needed', '3': 6, '4': 1, '5': 8, '10': 'reloadNeeded'},
    {'1': 'domain_risk', '3': 7, '4': 1, '5': 8, '10': 'domainRisk'},
    {'1': 'cache_bytes', '3': 8, '4': 1, '5': 3, '10': 'cacheBytes'},
    {'1': 'pins', '3': 9, '4': 3, '5': 9, '10': 'pins'},
    {'1': 'device_id', '3': 10, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'vault_version', '3': 11, '4': 1, '5': 4, '10': 'vaultVersion'},
    {'1': 'files_bytes', '3': 12, '4': 1, '5': 3, '10': 'filesBytes'},
    {
      '1': 'writable_bytes',
      '3': 13,
      '4': 1,
      '5': 3,
      '9': 0,
      '10': 'writableBytes',
      '17': true
    },
    {'1': 'pinned_bytes', '3': 14, '4': 1, '5': 3, '10': 'pinnedBytes'},
    {
      '1': 'pinned_missing_bytes',
      '3': 15,
      '4': 1,
      '5': 3,
      '10': 'pinnedMissingBytes'
    },
    {'1': 'pinning', '3': 16, '4': 1, '5': 8, '10': 'pinning'},
    {'1': 'pin_error', '3': 17, '4': 1, '5': 9, '10': 'pinError'},
  ],
  '8': [
    {'1': '_writable_bytes'},
  ],
};

/// Descriptor for `StatusReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List statusReplyDescriptor = $convert.base64Decode(
    'CgtTdGF0dXNSZXBseRIfCgtkYXRhX3NoYXJkcxgBIAEoBVIKZGF0YVNoYXJkcxIjCg1wYXJpdH'
    'lfc2hhcmRzGAIgASgFUgxwYXJpdHlTaGFyZHMSNgoIYmFja2VuZHMYAyADKAsyGi5zZWxmaG9s'
    'ZC5zaGFyZC52MS5CYWNrZW5kUghiYWNrZW5kcxIzCgdkZXZpY2VzGAQgAygLMhkuc2VsZmhvbG'
    'Quc2hhcmQudjEuRGV2aWNlUgdkZXZpY2VzEhgKB3BlbmRpbmcYBSABKAVSB3BlbmRpbmcSIwoN'
    'cmVsb2FkX25lZWRlZBgGIAEoCFIMcmVsb2FkTmVlZGVkEh8KC2RvbWFpbl9yaXNrGAcgASgIUg'
    'pkb21haW5SaXNrEh8KC2NhY2hlX2J5dGVzGAggASgDUgpjYWNoZUJ5dGVzEhIKBHBpbnMYCSAD'
    'KAlSBHBpbnMSGwoJZGV2aWNlX2lkGAogASgJUghkZXZpY2VJZBIjCg12YXVsdF92ZXJzaW9uGA'
    'sgASgEUgx2YXVsdFZlcnNpb24SHwoLZmlsZXNfYnl0ZXMYDCABKANSCmZpbGVzQnl0ZXMSKgoO'
    'd3JpdGFibGVfYnl0ZXMYDSABKANIAFINd3JpdGFibGVCeXRlc4gBARIhCgxwaW5uZWRfYnl0ZX'
    'MYDiABKANSC3Bpbm5lZEJ5dGVzEjAKFHBpbm5lZF9taXNzaW5nX2J5dGVzGA8gASgDUhJwaW5u'
    'ZWRNaXNzaW5nQnl0ZXMSGAoHcGlubmluZxgQIAEoCFIHcGlubmluZxIbCglwaW5fZXJyb3IYES'
    'ABKAlSCHBpbkVycm9yQhEKD193cml0YWJsZV9ieXRlcw==');

@$core.Deprecated('Use pathRequestDescriptor instead')
const PathRequest$json = {
  '1': 'PathRequest',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
  ],
};

/// Descriptor for `PathRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pathRequestDescriptor =
    $convert.base64Decode('CgtQYXRoUmVxdWVzdBISCgRwYXRoGAEgASgJUgRwYXRo');

@$core.Deprecated('Use pathsRequestDescriptor instead')
const PathsRequest$json = {
  '1': 'PathsRequest',
  '2': [
    {'1': 'paths', '3': 1, '4': 3, '5': 9, '10': 'paths'},
  ],
};

/// Descriptor for `PathsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pathsRequestDescriptor =
    $convert.base64Decode('CgxQYXRoc1JlcXVlc3QSFAoFcGF0aHMYASADKAlSBXBhdGhz');

@$core.Deprecated('Use searchRequestDescriptor instead')
const SearchRequest$json = {
  '1': 'SearchRequest',
  '2': [
    {'1': 'query', '3': 1, '4': 1, '5': 9, '10': 'query'},
    {'1': 'limit', '3': 2, '4': 1, '5': 5, '10': 'limit'},
  ],
};

/// Descriptor for `SearchRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List searchRequestDescriptor = $convert.base64Decode(
    'Cg1TZWFyY2hSZXF1ZXN0EhQKBXF1ZXJ5GAEgASgJUgVxdWVyeRIUCgVsaW1pdBgCIAEoBVIFbG'
    'ltaXQ=');

@$core.Deprecated('Use entryDescriptor instead')
const Entry$json = {
  '1': 'Entry',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'dir', '3': 3, '4': 1, '5': 8, '10': 'dir'},
    {'1': 'size', '3': 4, '4': 1, '5': 3, '10': 'size'},
    {'1': 'mtime_unix', '3': 5, '4': 1, '5': 3, '10': 'mtimeUnix'},
    {'1': 'local', '3': 6, '4': 1, '5': 8, '10': 'local'},
    {'1': 'pinned', '3': 7, '4': 1, '5': 8, '10': 'pinned'},
    {'1': 'device', '3': 8, '4': 1, '5': 9, '10': 'device'},
    {'1': 'conflict_of', '3': 9, '4': 1, '5': 9, '10': 'conflictOf'},
  ],
};

/// Descriptor for `Entry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List entryDescriptor = $convert.base64Decode(
    'CgVFbnRyeRISCgRwYXRoGAEgASgJUgRwYXRoEhIKBG5hbWUYAiABKAlSBG5hbWUSEAoDZGlyGA'
    'MgASgIUgNkaXISEgoEc2l6ZRgEIAEoA1IEc2l6ZRIdCgptdGltZV91bml4GAUgASgDUgltdGlt'
    'ZVVuaXgSFAoFbG9jYWwYBiABKAhSBWxvY2FsEhYKBnBpbm5lZBgHIAEoCFIGcGlubmVkEhYKBm'
    'RldmljZRgIIAEoCVIGZGV2aWNlEh8KC2NvbmZsaWN0X29mGAkgASgJUgpjb25mbGljdE9m');

@$core.Deprecated('Use listReplyDescriptor instead')
const ListReply$json = {
  '1': 'ListReply',
  '2': [
    {
      '1': 'entries',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.Entry',
      '10': 'entries'
    },
  ],
};

/// Descriptor for `ListReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listReplyDescriptor = $convert.base64Decode(
    'CglMaXN0UmVwbHkSMgoHZW50cmllcxgBIAMoCzIYLnNlbGZob2xkLnNoYXJkLnYxLkVudHJ5Ug'
    'dlbnRyaWVz');

@$core.Deprecated('Use removeRequestDescriptor instead')
const RemoveRequest$json = {
  '1': 'RemoveRequest',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {'1': 'recursive', '3': 2, '4': 1, '5': 8, '10': 'recursive'},
  ],
};

/// Descriptor for `RemoveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeRequestDescriptor = $convert.base64Decode(
    'Cg1SZW1vdmVSZXF1ZXN0EhIKBHBhdGgYASABKAlSBHBhdGgSHAoJcmVjdXJzaXZlGAIgASgIUg'
    'lyZWN1cnNpdmU=');

@$core.Deprecated('Use renameRequestDescriptor instead')
const RenameRequest$json = {
  '1': 'RenameRequest',
  '2': [
    {'1': 'from', '3': 1, '4': 1, '5': 9, '10': 'from'},
    {'1': 'to', '3': 2, '4': 1, '5': 9, '10': 'to'},
  ],
};

/// Descriptor for `RenameRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List renameRequestDescriptor = $convert.base64Decode(
    'Cg1SZW5hbWVSZXF1ZXN0EhIKBGZyb20YASABKAlSBGZyb20SDgoCdG8YAiABKAlSAnRv');

@$core.Deprecated('Use uploadRequestDescriptor instead')
const UploadRequest$json = {
  '1': 'UploadRequest',
  '2': [
    {'1': 'local_paths', '3': 1, '4': 3, '5': 9, '10': 'localPaths'},
    {'1': 'remote_dir', '3': 2, '4': 1, '5': 9, '10': 'remoteDir'},
  ],
};

/// Descriptor for `UploadRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadRequestDescriptor = $convert.base64Decode(
    'Cg1VcGxvYWRSZXF1ZXN0Eh8KC2xvY2FsX3BhdGhzGAEgAygJUgpsb2NhbFBhdGhzEh0KCnJlbW'
    '90ZV9kaXIYAiABKAlSCXJlbW90ZURpcg==');

@$core.Deprecated('Use uploadReplyDescriptor instead')
const UploadReply$json = {
  '1': 'UploadReply',
  '2': [
    {'1': 'files', '3': 1, '4': 1, '5': 5, '10': 'files'},
  ],
};

/// Descriptor for `UploadReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadReplyDescriptor =
    $convert.base64Decode('CgtVcGxvYWRSZXBseRIUCgVmaWxlcxgBIAEoBVIFZmlsZXM=');

@$core.Deprecated('Use downloadRequestDescriptor instead')
const DownloadRequest$json = {
  '1': 'DownloadRequest',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {'1': 'local_path', '3': 2, '4': 1, '5': 9, '10': 'localPath'},
  ],
};

/// Descriptor for `DownloadRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List downloadRequestDescriptor = $convert.base64Decode(
    'Cg9Eb3dubG9hZFJlcXVlc3QSEgoEcGF0aBgBIAEoCVIEcGF0aBIdCgpsb2NhbF9wYXRoGAIgAS'
    'gJUglsb2NhbFBhdGg=');

@$core.Deprecated('Use lockInfoDescriptor instead')
const LockInfo$json = {
  '1': 'LockInfo',
  '2': [
    {'1': 'path', '3': 6, '4': 1, '5': 9, '10': 'path'},
    {'1': 'locked', '3': 1, '4': 1, '5': 8, '10': 'locked'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'device_name', '3': 3, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'since_unix', '3': 4, '4': 1, '5': 3, '10': 'sinceUnix'},
    {'1': 'mine', '3': 5, '4': 1, '5': 8, '10': 'mine'},
  ],
};

/// Descriptor for `LockInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List lockInfoDescriptor = $convert.base64Decode(
    'CghMb2NrSW5mbxISCgRwYXRoGAYgASgJUgRwYXRoEhYKBmxvY2tlZBgBIAEoCFIGbG9ja2VkEh'
    'sKCWRldmljZV9pZBgCIAEoCVIIZGV2aWNlSWQSHwoLZGV2aWNlX25hbWUYAyABKAlSCmRldmlj'
    'ZU5hbWUSHQoKc2luY2VfdW5peBgEIAEoA1IJc2luY2VVbml4EhIKBG1pbmUYBSABKAhSBG1pbm'
    'U=');

@$core.Deprecated('Use locksReplyDescriptor instead')
const LocksReply$json = {
  '1': 'LocksReply',
  '2': [
    {
      '1': 'locks',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.LockInfo',
      '10': 'locks'
    },
  ],
};

/// Descriptor for `LocksReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List locksReplyDescriptor = $convert.base64Decode(
    'CgpMb2Nrc1JlcGx5EjEKBWxvY2tzGAEgAygLMhsuc2VsZmhvbGQuc2hhcmQudjEuTG9ja0luZm'
    '9SBWxvY2tz');

@$core.Deprecated('Use repairRequestDescriptor instead')
const RepairRequest$json = {
  '1': 'RepairRequest',
  '2': [
    {'1': 'rebalance', '3': 1, '4': 1, '5': 8, '10': 'rebalance'},
  ],
};

/// Descriptor for `RepairRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List repairRequestDescriptor = $convert.base64Decode(
    'Cg1SZXBhaXJSZXF1ZXN0EhwKCXJlYmFsYW5jZRgBIAEoCFIJcmViYWxhbmNl');

@$core.Deprecated('Use repairReplyDescriptor instead')
const RepairReply$json = {
  '1': 'RepairReply',
  '2': [
    {'1': 'blobs', '3': 1, '4': 1, '5': 5, '10': 'blobs'},
    {'1': 'repaired', '3': 2, '4': 1, '5': 5, '10': 'repaired'},
    {'1': 'failed', '3': 3, '4': 1, '5': 5, '10': 'failed'},
    {'1': 'journal_copies', '3': 4, '4': 1, '5': 5, '10': 'journalCopies'},
    {'1': 'reencoded', '3': 5, '4': 1, '5': 5, '10': 'reencoded'},
    {'1': 'moved', '3': 6, '4': 1, '5': 5, '10': 'moved'},
    {'1': 'damaged', '3': 7, '4': 3, '5': 9, '10': 'damaged'},
    {'1': 'from_local', '3': 8, '4': 1, '5': 5, '10': 'fromLocal'},
  ],
};

/// Descriptor for `RepairReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List repairReplyDescriptor = $convert.base64Decode(
    'CgtSZXBhaXJSZXBseRIUCgVibG9icxgBIAEoBVIFYmxvYnMSGgoIcmVwYWlyZWQYAiABKAVSCH'
    'JlcGFpcmVkEhYKBmZhaWxlZBgDIAEoBVIGZmFpbGVkEiUKDmpvdXJuYWxfY29waWVzGAQgASgF'
    'Ug1qb3VybmFsQ29waWVzEhwKCXJlZW5jb2RlZBgFIAEoBVIJcmVlbmNvZGVkEhQKBW1vdmVkGA'
    'YgASgFUgVtb3ZlZBIYCgdkYW1hZ2VkGAcgAygJUgdkYW1hZ2VkEh0KCmZyb21fbG9jYWwYCCAB'
    'KAVSCWZyb21Mb2NhbA==');

@$core.Deprecated('Use gCRequestDescriptor instead')
const GCRequest$json = {
  '1': 'GCRequest',
  '2': [
    {'1': 'grace_seconds', '3': 1, '4': 1, '5': 3, '10': 'graceSeconds'},
  ],
};

/// Descriptor for `GCRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gCRequestDescriptor = $convert.base64Decode(
    'CglHQ1JlcXVlc3QSIwoNZ3JhY2Vfc2Vjb25kcxgBIAEoA1IMZ3JhY2VTZWNvbmRz');

@$core.Deprecated('Use gCReplyDescriptor instead')
const GCReply$json = {
  '1': 'GCReply',
  '2': [
    {'1': 'objects', '3': 1, '4': 1, '5': 5, '10': 'objects'},
    {'1': 'unused', '3': 2, '4': 1, '5': 5, '10': 'unused'},
    {'1': 'deleted', '3': 3, '4': 1, '5': 5, '10': 'deleted'},
    {'1': 'skipped', '3': 4, '4': 3, '5': 9, '10': 'skipped'},
  ],
};

/// Descriptor for `GCReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gCReplyDescriptor = $convert.base64Decode(
    'CgdHQ1JlcGx5EhgKB29iamVjdHMYASABKAVSB29iamVjdHMSFgoGdW51c2VkGAIgASgFUgZ1bn'
    'VzZWQSGAoHZGVsZXRlZBgDIAEoBVIHZGVsZXRlZBIYCgdza2lwcGVkGAQgAygJUgdza2lwcGVk');

@$core.Deprecated('Use setRedundancyRequestDescriptor instead')
const SetRedundancyRequest$json = {
  '1': 'SetRedundancyRequest',
  '2': [
    {'1': 'data_shards', '3': 1, '4': 1, '5': 5, '10': 'dataShards'},
    {'1': 'parity_shards', '3': 2, '4': 1, '5': 5, '10': 'parityShards'},
  ],
};

/// Descriptor for `SetRedundancyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setRedundancyRequestDescriptor = $convert.base64Decode(
    'ChRTZXRSZWR1bmRhbmN5UmVxdWVzdBIfCgtkYXRhX3NoYXJkcxgBIAEoBVIKZGF0YVNoYXJkcx'
    'IjCg1wYXJpdHlfc2hhcmRzGAIgASgFUgxwYXJpdHlTaGFyZHM=');

@$core.Deprecated('Use eventDescriptor instead')
const Event$json = {
  '1': 'Event',
  '2': [
    {
      '1': 'kind',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.Event.Kind',
      '10': 'kind'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
  '4': [Event_Kind$json],
};

@$core.Deprecated('Use eventDescriptor instead')
const Event_Kind$json = {
  '1': 'Kind',
  '2': [
    {'1': 'KIND_UNSPECIFIED', '2': 0},
    {'1': 'TREE_CHANGED', '2': 1},
    {'1': 'STATE_CHANGED', '2': 2},
    {'1': 'SYNC_DONE', '2': 3},
    {'1': 'SYNC_ERROR', '2': 4},
    {'1': 'SECRETS_CHANGED', '2': 5},
    {'1': 'APPS_CHANGED', '2': 6},
  ],
};

/// Descriptor for `Event`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eventDescriptor = $convert.base64Decode(
    'CgVFdmVudBIxCgRraW5kGAEgASgOMh0uc2VsZmhvbGQuc2hhcmQudjEuRXZlbnQuS2luZFIEa2'
    'luZBIYCgdtZXNzYWdlGAIgASgJUgdtZXNzYWdlIocBCgRLaW5kEhQKEEtJTkRfVU5TUEVDSUZJ'
    'RUQQABIQCgxUUkVFX0NIQU5HRUQQARIRCg1TVEFURV9DSEFOR0VEEAISDQoJU1lOQ19ET05FEA'
    'MSDgoKU1lOQ19FUlJPUhAEEhMKD1NFQ1JFVFNfQ0hBTkdFRBAFEhAKDEFQUFNfQ0hBTkdFRBAG');

@$core.Deprecated('Use logsRequestDescriptor instead')
const LogsRequest$json = {
  '1': 'LogsRequest',
  '2': [
    {'1': 'backend_id', '3': 1, '4': 1, '5': 9, '10': 'backendId'},
    {'1': 'limit', '3': 2, '4': 1, '5': 5, '10': 'limit'},
  ],
};

/// Descriptor for `LogsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logsRequestDescriptor = $convert.base64Decode(
    'CgtMb2dzUmVxdWVzdBIdCgpiYWNrZW5kX2lkGAEgASgJUgliYWNrZW5kSWQSFAoFbGltaXQYAi'
    'ABKAVSBWxpbWl0');

@$core.Deprecated('Use logEntryDescriptor instead')
const LogEntry$json = {
  '1': 'LogEntry',
  '2': [
    {'1': 'seq', '3': 1, '4': 1, '5': 4, '10': 'seq'},
    {'1': 'time_unix_ms', '3': 2, '4': 1, '5': 3, '10': 'timeUnixMs'},
    {'1': 'first_unix_ms', '3': 3, '4': 1, '5': 3, '10': 'firstUnixMs'},
    {'1': 'level', '3': 4, '4': 1, '5': 9, '10': 'level'},
    {'1': 'source', '3': 5, '4': 1, '5': 9, '10': 'source'},
    {'1': 'backend_id', '3': 6, '4': 1, '5': 9, '10': 'backendId'},
    {'1': 'message', '3': 7, '4': 1, '5': 9, '10': 'message'},
    {'1': 'count', '3': 8, '4': 1, '5': 5, '10': 'count'},
  ],
};

/// Descriptor for `LogEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logEntryDescriptor = $convert.base64Decode(
    'CghMb2dFbnRyeRIQCgNzZXEYASABKARSA3NlcRIgCgx0aW1lX3VuaXhfbXMYAiABKANSCnRpbW'
    'VVbml4TXMSIgoNZmlyc3RfdW5peF9tcxgDIAEoA1ILZmlyc3RVbml4TXMSFAoFbGV2ZWwYBCAB'
    'KAlSBWxldmVsEhYKBnNvdXJjZRgFIAEoCVIGc291cmNlEh0KCmJhY2tlbmRfaWQYBiABKAlSCW'
    'JhY2tlbmRJZBIYCgdtZXNzYWdlGAcgASgJUgdtZXNzYWdlEhQKBWNvdW50GAggASgFUgVjb3Vu'
    'dA==');

@$core.Deprecated('Use logsReplyDescriptor instead')
const LogsReply$json = {
  '1': 'LogsReply',
  '2': [
    {
      '1': 'entries',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.LogEntry',
      '10': 'entries'
    },
  ],
};

/// Descriptor for `LogsReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logsReplyDescriptor = $convert.base64Decode(
    'CglMb2dzUmVwbHkSNQoHZW50cmllcxgBIAMoCzIbLnNlbGZob2xkLnNoYXJkLnYxLkxvZ0VudH'
    'J5UgdlbnRyaWVz');

@$core.Deprecated('Use secretEntryDescriptor instead')
const SecretEntry$json = {
  '1': 'SecretEntry',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'version', '3': 2, '4': 1, '5': 9, '10': 'version'},
    {
      '1': 'kind',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.SecretKind',
      '10': 'kind'
    },
    {'1': 'title', '3': 4, '4': 1, '5': 9, '10': 'title'},
    {'1': 'group_id', '3': 5, '4': 1, '5': 9, '10': 'groupId'},
    {'1': 'username', '3': 6, '4': 1, '5': 9, '10': 'username'},
    {'1': 'url', '3': 7, '4': 1, '5': 9, '10': 'url'},
    {'1': 'comment', '3': 8, '4': 1, '5': 9, '10': 'comment'},
    {
      '1': 'network',
      '3': 9,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.Web3Network',
      '10': 'network'
    },
    {'1': 'key_type', '3': 10, '4': 1, '5': 9, '10': 'keyType'},
    {'1': 'public_key', '3': 11, '4': 1, '5': 9, '10': 'publicKey'},
    {'1': 'fingerprint', '3': 12, '4': 1, '5': 9, '10': 'fingerprint'},
    {'1': 'address', '3': 13, '4': 1, '5': 9, '10': 'address'},
    {'1': 'created_unix', '3': 14, '4': 1, '5': 3, '10': 'createdUnix'},
    {'1': 'updated_unix', '3': 15, '4': 1, '5': 3, '10': 'updatedUnix'},
    {'1': 'device', '3': 16, '4': 1, '5': 9, '10': 'device'},
    {'1': 'conflict_of', '3': 17, '4': 1, '5': 9, '10': 'conflictOf'},
    {'1': 'recovery_left', '3': 18, '4': 1, '5': 5, '10': 'recoveryLeft'},
    {'1': 'opened', '3': 20, '4': 1, '5': 8, '10': 'opened'},
    {'1': 'password', '3': 21, '4': 1, '5': 9, '10': 'password'},
    {'1': 'notes', '3': 22, '4': 1, '5': 9, '10': 'notes'},
    {'1': 'private_key', '3': 23, '4': 1, '5': 9, '10': 'privateKey'},
    {'1': 'passphrase', '3': 24, '4': 1, '5': 9, '10': 'passphrase'},
    {'1': 'totp_secret', '3': 25, '4': 1, '5': 9, '10': 'totpSecret'},
    {'1': 'totp_algorithm', '3': 26, '4': 1, '5': 9, '10': 'totpAlgorithm'},
    {'1': 'totp_digits', '3': 27, '4': 1, '5': 5, '10': 'totpDigits'},
    {'1': 'totp_period', '3': 28, '4': 1, '5': 5, '10': 'totpPeriod'},
    {'1': 'recovery_codes', '3': 29, '4': 3, '5': 9, '10': 'recoveryCodes'},
    {
      '1': 'server',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.selfhold.shard.v1.SecretServer',
      '10': 'server'
    },
  ],
};

/// Descriptor for `SecretEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretEntryDescriptor = $convert.base64Decode(
    'CgtTZWNyZXRFbnRyeRIOCgJpZBgBIAEoCVICaWQSGAoHdmVyc2lvbhgCIAEoCVIHdmVyc2lvbh'
    'IxCgRraW5kGAMgASgOMh0uc2VsZmhvbGQuc2hhcmQudjEuU2VjcmV0S2luZFIEa2luZBIUCgV0'
    'aXRsZRgEIAEoCVIFdGl0bGUSGQoIZ3JvdXBfaWQYBSABKAlSB2dyb3VwSWQSGgoIdXNlcm5hbW'
    'UYBiABKAlSCHVzZXJuYW1lEhAKA3VybBgHIAEoCVIDdXJsEhgKB2NvbW1lbnQYCCABKAlSB2Nv'
    'bW1lbnQSOAoHbmV0d29yaxgJIAEoDjIeLnNlbGZob2xkLnNoYXJkLnYxLldlYjNOZXR3b3JrUg'
    'duZXR3b3JrEhkKCGtleV90eXBlGAogASgJUgdrZXlUeXBlEh0KCnB1YmxpY19rZXkYCyABKAlS'
    'CXB1YmxpY0tleRIgCgtmaW5nZXJwcmludBgMIAEoCVILZmluZ2VycHJpbnQSGAoHYWRkcmVzcx'
    'gNIAEoCVIHYWRkcmVzcxIhCgxjcmVhdGVkX3VuaXgYDiABKANSC2NyZWF0ZWRVbml4EiEKDHVw'
    'ZGF0ZWRfdW5peBgPIAEoA1ILdXBkYXRlZFVuaXgSFgoGZGV2aWNlGBAgASgJUgZkZXZpY2USHw'
    'oLY29uZmxpY3Rfb2YYESABKAlSCmNvbmZsaWN0T2YSIwoNcmVjb3ZlcnlfbGVmdBgSIAEoBVIM'
    'cmVjb3ZlcnlMZWZ0EhYKBm9wZW5lZBgUIAEoCFIGb3BlbmVkEhoKCHBhc3N3b3JkGBUgASgJUg'
    'hwYXNzd29yZBIUCgVub3RlcxgWIAEoCVIFbm90ZXMSHwoLcHJpdmF0ZV9rZXkYFyABKAlSCnBy'
    'aXZhdGVLZXkSHgoKcGFzc3BocmFzZRgYIAEoCVIKcGFzc3BocmFzZRIfCgt0b3RwX3NlY3JldB'
    'gZIAEoCVIKdG90cFNlY3JldBIlCg50b3RwX2FsZ29yaXRobRgaIAEoCVINdG90cEFsZ29yaXRo'
    'bRIfCgt0b3RwX2RpZ2l0cxgbIAEoBVIKdG90cERpZ2l0cxIfCgt0b3RwX3BlcmlvZBgcIAEoBV'
    'IKdG90cFBlcmlvZBIlCg5yZWNvdmVyeV9jb2RlcxgdIAMoCVINcmVjb3ZlcnlDb2RlcxI3CgZz'
    'ZXJ2ZXIYHiABKAsyHy5zZWxmaG9sZC5zaGFyZC52MS5TZWNyZXRTZXJ2ZXJSBnNlcnZlcg==');

@$core.Deprecated('Use secretServerDescriptor instead')
const SecretServer$json = {
  '1': 'SecretServer',
  '2': [
    {'1': 'folder', '3': 1, '4': 1, '5': 9, '10': 'folder'},
    {
      '1': 'auth',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.ServerAuth',
      '10': 'auth'
    },
    {'1': 'ssh_key_id', '3': 3, '4': 1, '5': 9, '10': 'sshKeyId'},
    {'1': 'host_keys', '3': 4, '4': 3, '5': 9, '10': 'hostKeys'},
    {'1': 'jump_server_id', '3': 5, '4': 1, '5': 9, '10': 'jumpServerId'},
    {'1': 'tags', '3': 6, '4': 3, '5': 9, '10': 'tags'},
    {'1': 'extra_keys', '3': 7, '4': 3, '5': 9, '10': 'extraKeys'},
  ],
};

/// Descriptor for `SecretServer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretServerDescriptor = $convert.base64Decode(
    'CgxTZWNyZXRTZXJ2ZXISFgoGZm9sZGVyGAEgASgJUgZmb2xkZXISMQoEYXV0aBgCIAEoDjIdLn'
    'NlbGZob2xkLnNoYXJkLnYxLlNlcnZlckF1dGhSBGF1dGgSHAoKc3NoX2tleV9pZBgDIAEoCVII'
    'c3NoS2V5SWQSGwoJaG9zdF9rZXlzGAQgAygJUghob3N0S2V5cxIkCg5qdW1wX3NlcnZlcl9pZB'
    'gFIAEoCVIManVtcFNlcnZlcklkEhIKBHRhZ3MYBiADKAlSBHRhZ3MSHQoKZXh0cmFfa2V5cxgH'
    'IAMoCVIJZXh0cmFLZXlz');

@$core.Deprecated('Use secretGroupDescriptor instead')
const SecretGroup$json = {
  '1': 'SecretGroup',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'icon', '3': 3, '4': 1, '5': 12, '10': 'icon'},
    {'1': 'conflict_of', '3': 4, '4': 1, '5': 9, '10': 'conflictOf'},
  ],
};

/// Descriptor for `SecretGroup`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretGroupDescriptor = $convert.base64Decode(
    'CgtTZWNyZXRHcm91cBIOCgJpZBgBIAEoCVICaWQSEgoEbmFtZRgCIAEoCVIEbmFtZRISCgRpY2'
    '9uGAMgASgMUgRpY29uEh8KC2NvbmZsaWN0X29mGAQgASgJUgpjb25mbGljdE9m');

@$core.Deprecated('Use secretsReplyDescriptor instead')
const SecretsReply$json = {
  '1': 'SecretsReply',
  '2': [
    {
      '1': 'groups',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.SecretGroup',
      '10': 'groups'
    },
    {
      '1': 'entries',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.SecretEntry',
      '10': 'entries'
    },
    {'1': 'broken', '3': 3, '4': 1, '5': 5, '10': 'broken'},
  ],
};

/// Descriptor for `SecretsReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretsReplyDescriptor = $convert.base64Decode(
    'CgxTZWNyZXRzUmVwbHkSNgoGZ3JvdXBzGAEgAygLMh4uc2VsZmhvbGQuc2hhcmQudjEuU2Vjcm'
    'V0R3JvdXBSBmdyb3VwcxI4CgdlbnRyaWVzGAIgAygLMh4uc2VsZmhvbGQuc2hhcmQudjEuU2Vj'
    'cmV0RW50cnlSB2VudHJpZXMSFgoGYnJva2VuGAMgASgFUgZicm9rZW4=');

@$core.Deprecated('Use deriveKeyRequestDescriptor instead')
const DeriveKeyRequest$json = {
  '1': 'DeriveKeyRequest',
  '2': [
    {
      '1': 'kind',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.SecretKind',
      '10': 'kind'
    },
    {
      '1': 'network',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.Web3Network',
      '10': 'network'
    },
    {'1': 'private_key', '3': 3, '4': 1, '5': 9, '10': 'privateKey'},
    {'1': 'passphrase', '3': 4, '4': 1, '5': 9, '10': 'passphrase'},
    {'1': 'comment', '3': 5, '4': 1, '5': 9, '10': 'comment'},
  ],
};

/// Descriptor for `DeriveKeyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deriveKeyRequestDescriptor = $convert.base64Decode(
    'ChBEZXJpdmVLZXlSZXF1ZXN0EjEKBGtpbmQYASABKA4yHS5zZWxmaG9sZC5zaGFyZC52MS5TZW'
    'NyZXRLaW5kUgRraW5kEjgKB25ldHdvcmsYAiABKA4yHi5zZWxmaG9sZC5zaGFyZC52MS5XZWIz'
    'TmV0d29ya1IHbmV0d29yaxIfCgtwcml2YXRlX2tleRgDIAEoCVIKcHJpdmF0ZUtleRIeCgpwYX'
    'NzcGhyYXNlGAQgASgJUgpwYXNzcGhyYXNlEhgKB2NvbW1lbnQYBSABKAlSB2NvbW1lbnQ=');

@$core.Deprecated('Use derivedKeyDescriptor instead')
const DerivedKey$json = {
  '1': 'DerivedKey',
  '2': [
    {'1': 'key_type', '3': 1, '4': 1, '5': 9, '10': 'keyType'},
    {'1': 'public_key', '3': 2, '4': 1, '5': 9, '10': 'publicKey'},
    {'1': 'fingerprint', '3': 3, '4': 1, '5': 9, '10': 'fingerprint'},
    {'1': 'address', '3': 4, '4': 1, '5': 9, '10': 'address'},
  ],
};

/// Descriptor for `DerivedKey`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List derivedKeyDescriptor = $convert.base64Decode(
    'CgpEZXJpdmVkS2V5EhkKCGtleV90eXBlGAEgASgJUgdrZXlUeXBlEh0KCnB1YmxpY19rZXkYAi'
    'ABKAlSCXB1YmxpY0tleRIgCgtmaW5nZXJwcmludBgDIAEoCVILZmluZ2VycHJpbnQSGAoHYWRk'
    'cmVzcxgEIAEoCVIHYWRkcmVzcw==');

@$core.Deprecated('Use totpCodeDescriptor instead')
const TotpCode$json = {
  '1': 'TotpCode',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'code', '3': 2, '4': 1, '5': 9, '10': 'code'},
    {'1': 'next', '3': 3, '4': 1, '5': 9, '10': 'next'},
    {'1': 'period', '3': 4, '4': 1, '5': 5, '10': 'period'},
    {'1': 'expires_unix', '3': 5, '4': 1, '5': 3, '10': 'expiresUnix'},
  ],
};

/// Descriptor for `TotpCode`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpCodeDescriptor = $convert.base64Decode(
    'CghUb3RwQ29kZRIOCgJpZBgBIAEoCVICaWQSEgoEY29kZRgCIAEoCVIEY29kZRISCgRuZXh0GA'
    'MgASgJUgRuZXh0EhYKBnBlcmlvZBgEIAEoBVIGcGVyaW9kEiEKDGV4cGlyZXNfdW5peBgFIAEo'
    'A1ILZXhwaXJlc1VuaXg=');

@$core.Deprecated('Use totpCodesReplyDescriptor instead')
const TotpCodesReply$json = {
  '1': 'TotpCodesReply',
  '2': [
    {
      '1': 'codes',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.TotpCode',
      '10': 'codes'
    },
    {'1': 'now_unix', '3': 2, '4': 1, '5': 3, '10': 'nowUnix'},
  ],
};

/// Descriptor for `TotpCodesReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpCodesReplyDescriptor = $convert.base64Decode(
    'Cg5Ub3RwQ29kZXNSZXBseRIxCgVjb2RlcxgBIAMoCzIbLnNlbGZob2xkLnNoYXJkLnYxLlRvdH'
    'BDb2RlUgVjb2RlcxIZCghub3dfdW5peBgCIAEoA1IHbm93VW5peA==');

@$core.Deprecated('Use recoveryCodeRequestDescriptor instead')
const RecoveryCodeRequest$json = {
  '1': 'RecoveryCodeRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'code', '3': 2, '4': 1, '5': 9, '10': 'code'},
  ],
};

/// Descriptor for `RecoveryCodeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List recoveryCodeRequestDescriptor = $convert.base64Decode(
    'ChNSZWNvdmVyeUNvZGVSZXF1ZXN0Eg4KAmlkGAEgASgJUgJpZBISCgRjb2RlGAIgASgJUgRjb2'
    'Rl');

@$core.Deprecated('Use checkTotpRequestDescriptor instead')
const CheckTotpRequest$json = {
  '1': 'CheckTotpRequest',
  '2': [
    {'1': 'secret', '3': 1, '4': 1, '5': 9, '10': 'secret'},
    {'1': 'algorithm', '3': 2, '4': 1, '5': 9, '10': 'algorithm'},
    {'1': 'digits', '3': 3, '4': 1, '5': 5, '10': 'digits'},
    {'1': 'period', '3': 4, '4': 1, '5': 5, '10': 'period'},
    {'1': 'recovery_text', '3': 5, '4': 1, '5': 9, '10': 'recoveryText'},
  ],
};

/// Descriptor for `CheckTotpRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List checkTotpRequestDescriptor = $convert.base64Decode(
    'ChBDaGVja1RvdHBSZXF1ZXN0EhYKBnNlY3JldBgBIAEoCVIGc2VjcmV0EhwKCWFsZ29yaXRobR'
    'gCIAEoCVIJYWxnb3JpdGhtEhYKBmRpZ2l0cxgDIAEoBVIGZGlnaXRzEhYKBnBlcmlvZBgEIAEo'
    'BVIGcGVyaW9kEiMKDXJlY292ZXJ5X3RleHQYBSABKAlSDHJlY292ZXJ5VGV4dA==');

@$core.Deprecated('Use totpPreviewDescriptor instead')
const TotpPreview$json = {
  '1': 'TotpPreview',
  '2': [
    {'1': 'issuer', '3': 1, '4': 1, '5': 9, '10': 'issuer'},
    {'1': 'account', '3': 2, '4': 1, '5': 9, '10': 'account'},
    {'1': 'algorithm', '3': 3, '4': 1, '5': 9, '10': 'algorithm'},
    {'1': 'digits', '3': 4, '4': 1, '5': 5, '10': 'digits'},
    {'1': 'period', '3': 5, '4': 1, '5': 5, '10': 'period'},
    {'1': 'code', '3': 6, '4': 1, '5': 9, '10': 'code'},
    {'1': 'recovery_codes', '3': 7, '4': 3, '5': 9, '10': 'recoveryCodes'},
    {'1': 'error', '3': 8, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `TotpPreview`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpPreviewDescriptor = $convert.base64Decode(
    'CgtUb3RwUHJldmlldxIWCgZpc3N1ZXIYASABKAlSBmlzc3VlchIYCgdhY2NvdW50GAIgASgJUg'
    'dhY2NvdW50EhwKCWFsZ29yaXRobRgDIAEoCVIJYWxnb3JpdGhtEhYKBmRpZ2l0cxgEIAEoBVIG'
    'ZGlnaXRzEhYKBnBlcmlvZBgFIAEoBVIGcGVyaW9kEhIKBGNvZGUYBiABKAlSBGNvZGUSJQoOcm'
    'Vjb3ZlcnlfY29kZXMYByADKAlSDXJlY292ZXJ5Q29kZXMSFAoFZXJyb3IYCCABKAlSBWVycm9y');

@$core.Deprecated('Use totpLinksRequestDescriptor instead')
const TotpLinksRequest$json = {
  '1': 'TotpLinksRequest',
  '2': [
    {'1': 'text', '3': 1, '4': 1, '5': 9, '10': 'text'},
  ],
};

/// Descriptor for `TotpLinksRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpLinksRequestDescriptor = $convert
    .base64Decode('ChBUb3RwTGlua3NSZXF1ZXN0EhIKBHRleHQYASABKAlSBHRleHQ=');

@$core.Deprecated('Use totpLinkDescriptor instead')
const TotpLink$json = {
  '1': 'TotpLink',
  '2': [
    {'1': 'link', '3': 1, '4': 1, '5': 9, '10': 'link'},
    {
      '1': 'preview',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.selfhold.shard.v1.TotpPreview',
      '10': 'preview'
    },
  ],
};

/// Descriptor for `TotpLink`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpLinkDescriptor = $convert.base64Decode(
    'CghUb3RwTGluaxISCgRsaW5rGAEgASgJUgRsaW5rEjgKB3ByZXZpZXcYAiABKAsyHi5zZWxmaG'
    '9sZC5zaGFyZC52MS5Ub3RwUHJldmlld1IHcHJldmlldw==');

@$core.Deprecated('Use totpBatchDescriptor instead')
const TotpBatch$json = {
  '1': 'TotpBatch',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 5, '10': 'id'},
    {'1': 'index', '3': 2, '4': 1, '5': 5, '10': 'index'},
    {'1': 'size', '3': 3, '4': 1, '5': 5, '10': 'size'},
  ],
};

/// Descriptor for `TotpBatch`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpBatchDescriptor = $convert.base64Decode(
    'CglUb3RwQmF0Y2gSDgoCaWQYASABKAVSAmlkEhQKBWluZGV4GAIgASgFUgVpbmRleBISCgRzaX'
    'plGAMgASgFUgRzaXpl');

@$core.Deprecated('Use totpLinksReplyDescriptor instead')
const TotpLinksReply$json = {
  '1': 'TotpLinksReply',
  '2': [
    {
      '1': 'links',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.TotpLink',
      '10': 'links'
    },
    {
      '1': 'batches',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.TotpBatch',
      '10': 'batches'
    },
  ],
};

/// Descriptor for `TotpLinksReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List totpLinksReplyDescriptor = $convert.base64Decode(
    'Cg5Ub3RwTGlua3NSZXBseRIxCgVsaW5rcxgBIAMoCzIbLnNlbGZob2xkLnNoYXJkLnYxLlRvdH'
    'BMaW5rUgVsaW5rcxI2CgdiYXRjaGVzGAIgAygLMhwuc2VsZmhvbGQuc2hhcmQudjEuVG90cEJh'
    'dGNoUgdiYXRjaGVz');

@$core.Deprecated('Use secretImportRequestDescriptor instead')
const SecretImportRequest$json = {
  '1': 'SecretImportRequest',
  '2': [
    {'1': 'data', '3': 1, '4': 1, '5': 12, '10': 'data'},
    {'1': 'file_name', '3': 2, '4': 1, '5': 9, '10': 'fileName'},
    {'1': 'password', '3': 3, '4': 1, '5': 9, '10': 'password'},
    {'1': 'key_file', '3': 4, '4': 1, '5': 12, '10': 'keyFile'},
    {'1': 'group_id', '3': 5, '4': 1, '5': 9, '10': 'groupId'},
    {'1': 'keep_groups', '3': 6, '4': 1, '5': 8, '10': 'keepGroups'},
    {'1': 'dry_run', '3': 7, '4': 1, '5': 8, '10': 'dryRun'},
  ],
};

/// Descriptor for `SecretImportRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretImportRequestDescriptor = $convert.base64Decode(
    'ChNTZWNyZXRJbXBvcnRSZXF1ZXN0EhIKBGRhdGEYASABKAxSBGRhdGESGwoJZmlsZV9uYW1lGA'
    'IgASgJUghmaWxlTmFtZRIaCghwYXNzd29yZBgDIAEoCVIIcGFzc3dvcmQSGQoIa2V5X2ZpbGUY'
    'BCABKAxSB2tleUZpbGUSGQoIZ3JvdXBfaWQYBSABKAlSB2dyb3VwSWQSHwoLa2VlcF9ncm91cH'
    'MYBiABKAhSCmtlZXBHcm91cHMSFwoHZHJ5X3J1bhgHIAEoCFIGZHJ5UnVu');

@$core.Deprecated('Use secretImportItemDescriptor instead')
const SecretImportItem$json = {
  '1': 'SecretImportItem',
  '2': [
    {
      '1': 'kind',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.SecretKind',
      '10': 'kind'
    },
    {'1': 'title', '3': 2, '4': 1, '5': 9, '10': 'title'},
    {'1': 'username', '3': 3, '4': 1, '5': 9, '10': 'username'},
    {'1': 'url', '3': 4, '4': 1, '5': 9, '10': 'url'},
    {'1': 'group', '3': 5, '4': 1, '5': 9, '10': 'group'},
    {
      '1': 'status',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.selfhold.shard.v1.SecretImportStatus',
      '10': 'status'
    },
    {'1': 'error', '3': 7, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `SecretImportItem`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretImportItemDescriptor = $convert.base64Decode(
    'ChBTZWNyZXRJbXBvcnRJdGVtEjEKBGtpbmQYASABKA4yHS5zZWxmaG9sZC5zaGFyZC52MS5TZW'
    'NyZXRLaW5kUgRraW5kEhQKBXRpdGxlGAIgASgJUgV0aXRsZRIaCgh1c2VybmFtZRgDIAEoCVII'
    'dXNlcm5hbWUSEAoDdXJsGAQgASgJUgN1cmwSFAoFZ3JvdXAYBSABKAlSBWdyb3VwEj0KBnN0YX'
    'R1cxgGIAEoDjIlLnNlbGZob2xkLnNoYXJkLnYxLlNlY3JldEltcG9ydFN0YXR1c1IGc3RhdHVz'
    'EhQKBWVycm9yGAcgASgJUgVlcnJvcg==');

@$core.Deprecated('Use secretImportReplyDescriptor instead')
const SecretImportReply$json = {
  '1': 'SecretImportReply',
  '2': [
    {'1': 'need_password', '3': 1, '4': 1, '5': 8, '10': 'needPassword'},
    {'1': 'format', '3': 2, '4': 1, '5': 9, '10': 'format'},
    {
      '1': 'items',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.SecretImportItem',
      '10': 'items'
    },
    {'1': 'added', '3': 4, '4': 1, '5': 5, '10': 'added'},
    {'1': 'duplicates', '3': 5, '4': 1, '5': 5, '10': 'duplicates'},
    {'1': 'failed', '3': 6, '4': 1, '5': 5, '10': 'failed'},
    {'1': 'new_groups', '3': 7, '4': 3, '5': 9, '10': 'newGroups'},
  ],
};

/// Descriptor for `SecretImportReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List secretImportReplyDescriptor = $convert.base64Decode(
    'ChFTZWNyZXRJbXBvcnRSZXBseRIjCg1uZWVkX3Bhc3N3b3JkGAEgASgIUgxuZWVkUGFzc3dvcm'
    'QSFgoGZm9ybWF0GAIgASgJUgZmb3JtYXQSOQoFaXRlbXMYAyADKAsyIy5zZWxmaG9sZC5zaGFy'
    'ZC52MS5TZWNyZXRJbXBvcnRJdGVtUgVpdGVtcxIUCgVhZGRlZBgEIAEoBVIFYWRkZWQSHgoKZH'
    'VwbGljYXRlcxgFIAEoBVIKZHVwbGljYXRlcxIWCgZmYWlsZWQYBiABKAVSBmZhaWxlZBIdCgpu'
    'ZXdfZ3JvdXBzGAcgAygJUgluZXdHcm91cHM=');

@$core.Deprecated('Use pairingRequestInfoDescriptor instead')
const PairingRequestInfo$json = {
  '1': 'PairingRequestInfo',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'app_id', '3': 2, '4': 1, '5': 9, '10': 'appId'},
    {'1': 'app_name', '3': 3, '4': 1, '5': 9, '10': 'appName'},
    {'1': 'description', '3': 4, '4': 1, '5': 9, '10': 'description'},
    {'1': 'code', '3': 5, '4': 1, '5': 9, '10': 'code'},
    {'1': 'platform', '3': 6, '4': 1, '5': 9, '10': 'platform'},
    {'1': 'package', '3': 7, '4': 1, '5': 9, '10': 'package'},
    {'1': 'cert', '3': 8, '4': 1, '5': 9, '10': 'cert'},
    {'1': 'created_unix_ms', '3': 9, '4': 1, '5': 3, '10': 'createdUnixMs'},
    {'1': 'expires_unix_ms', '3': 10, '4': 1, '5': 3, '10': 'expiresUnixMs'},
    {'1': 'already_paired', '3': 11, '4': 1, '5': 8, '10': 'alreadyPaired'},
    {'1': 'scopes', '3': 12, '4': 3, '5': 9, '10': 'scopes'},
  ],
};

/// Descriptor for `PairingRequestInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingRequestInfoDescriptor = $convert.base64Decode(
    'ChJQYWlyaW5nUmVxdWVzdEluZm8SDgoCaWQYASABKAlSAmlkEhUKBmFwcF9pZBgCIAEoCVIFYX'
    'BwSWQSGQoIYXBwX25hbWUYAyABKAlSB2FwcE5hbWUSIAoLZGVzY3JpcHRpb24YBCABKAlSC2Rl'
    'c2NyaXB0aW9uEhIKBGNvZGUYBSABKAlSBGNvZGUSGgoIcGxhdGZvcm0YBiABKAlSCHBsYXRmb3'
    'JtEhgKB3BhY2thZ2UYByABKAlSB3BhY2thZ2USEgoEY2VydBgIIAEoCVIEY2VydBImCg9jcmVh'
    'dGVkX3VuaXhfbXMYCSABKANSDWNyZWF0ZWRVbml4TXMSJgoPZXhwaXJlc191bml4X21zGAogAS'
    'gDUg1leHBpcmVzVW5peE1zEiUKDmFscmVhZHlfcGFpcmVkGAsgASgIUg1hbHJlYWR5UGFpcmVk'
    'EhYKBnNjb3BlcxgMIAMoCVIGc2NvcGVz');

@$core.Deprecated('Use appPairingDescriptor instead')
const AppPairing$json = {
  '1': 'AppPairing',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'app_id', '3': 2, '4': 1, '5': 9, '10': 'appId'},
    {'1': 'app_name', '3': 3, '4': 1, '5': 9, '10': 'appName'},
    {'1': 'description', '3': 4, '4': 1, '5': 9, '10': 'description'},
    {'1': 'platform', '3': 5, '4': 1, '5': 9, '10': 'platform'},
    {'1': 'package', '3': 6, '4': 1, '5': 9, '10': 'package'},
    {'1': 'cert', '3': 7, '4': 1, '5': 9, '10': 'cert'},
    {'1': 'approved_unix_ms', '3': 8, '4': 1, '5': 3, '10': 'approvedUnixMs'},
    {'1': 'last_used_unix_ms', '3': 9, '4': 1, '5': 3, '10': 'lastUsedUnixMs'},
    {'1': 'scopes', '3': 10, '4': 3, '5': 9, '10': 'scopes'},
    {
      '1': 'secrets_read_unix_ms',
      '3': 11,
      '4': 1,
      '5': 3,
      '10': 'secretsReadUnixMs'
    },
    {'1': 'keys_read_unix_ms', '3': 12, '4': 1, '5': 3, '10': 'keysReadUnixMs'},
    {
      '1': 'servers_read_unix_ms',
      '3': 13,
      '4': 1,
      '5': 3,
      '10': 'serversReadUnixMs'
    },
  ],
};

/// Descriptor for `AppPairing`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List appPairingDescriptor = $convert.base64Decode(
    'CgpBcHBQYWlyaW5nEg4KAmlkGAEgASgJUgJpZBIVCgZhcHBfaWQYAiABKAlSBWFwcElkEhkKCG'
    'FwcF9uYW1lGAMgASgJUgdhcHBOYW1lEiAKC2Rlc2NyaXB0aW9uGAQgASgJUgtkZXNjcmlwdGlv'
    'bhIaCghwbGF0Zm9ybRgFIAEoCVIIcGxhdGZvcm0SGAoHcGFja2FnZRgGIAEoCVIHcGFja2FnZR'
    'ISCgRjZXJ0GAcgASgJUgRjZXJ0EigKEGFwcHJvdmVkX3VuaXhfbXMYCCABKANSDmFwcHJvdmVk'
    'VW5peE1zEikKEWxhc3RfdXNlZF91bml4X21zGAkgASgDUg5sYXN0VXNlZFVuaXhNcxIWCgZzY2'
    '9wZXMYCiADKAlSBnNjb3BlcxIvChRzZWNyZXRzX3JlYWRfdW5peF9tcxgLIAEoA1IRc2VjcmV0'
    'c1JlYWRVbml4TXMSKQoRa2V5c19yZWFkX3VuaXhfbXMYDCABKANSDmtleXNSZWFkVW5peE1zEi'
    '8KFHNlcnZlcnNfcmVhZF91bml4X21zGA0gASgDUhFzZXJ2ZXJzUmVhZFVuaXhNcw==');

@$core.Deprecated('Use appDataInfoDescriptor instead')
const AppDataInfo$json = {
  '1': 'AppDataInfo',
  '2': [
    {'1': 'app_id', '3': 1, '4': 1, '5': 9, '10': 'appId'},
    {'1': 'records', '3': 2, '4': 1, '5': 5, '10': 'records'},
    {'1': 'bytes', '3': 3, '4': 1, '5': 3, '10': 'bytes'},
  ],
};

/// Descriptor for `AppDataInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List appDataInfoDescriptor = $convert.base64Decode(
    'CgtBcHBEYXRhSW5mbxIVCgZhcHBfaWQYASABKAlSBWFwcElkEhgKB3JlY29yZHMYAiABKAVSB3'
    'JlY29yZHMSFAoFYnl0ZXMYAyABKANSBWJ5dGVz');

@$core.Deprecated('Use appsReplyDescriptor instead')
const AppsReply$json = {
  '1': 'AppsReply',
  '2': [
    {
      '1': 'requests',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.PairingRequestInfo',
      '10': 'requests'
    },
    {
      '1': 'pairings',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.AppPairing',
      '10': 'pairings'
    },
    {
      '1': 'data',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.selfhold.shard.v1.AppDataInfo',
      '10': 'data'
    },
  ],
};

/// Descriptor for `AppsReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List appsReplyDescriptor = $convert.base64Decode(
    'CglBcHBzUmVwbHkSQQoIcmVxdWVzdHMYASADKAsyJS5zZWxmaG9sZC5zaGFyZC52MS5QYWlyaW'
    '5nUmVxdWVzdEluZm9SCHJlcXVlc3RzEjkKCHBhaXJpbmdzGAIgAygLMh0uc2VsZmhvbGQuc2hh'
    'cmQudjEuQXBwUGFpcmluZ1IIcGFpcmluZ3MSMgoEZGF0YRgDIAMoCzIeLnNlbGZob2xkLnNoYX'
    'JkLnYxLkFwcERhdGFJbmZvUgRkYXRh');

@$core.Deprecated('Use decidePairingRequestDescriptor instead')
const DecidePairingRequest$json = {
  '1': 'DecidePairingRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'approve', '3': 2, '4': 1, '5': 8, '10': 'approve'},
    {'1': 'scopes', '3': 3, '4': 3, '5': 9, '10': 'scopes'},
  ],
};

/// Descriptor for `DecidePairingRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List decidePairingRequestDescriptor = $convert.base64Decode(
    'ChREZWNpZGVQYWlyaW5nUmVxdWVzdBIOCgJpZBgBIAEoCVICaWQSGAoHYXBwcm92ZRgCIAEoCF'
    'IHYXBwcm92ZRIWCgZzY29wZXMYAyADKAlSBnNjb3Blcw==');

@$core.Deprecated('Use setAppScopesRequestDescriptor instead')
const SetAppScopesRequest$json = {
  '1': 'SetAppScopesRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'scopes', '3': 2, '4': 3, '5': 9, '10': 'scopes'},
  ],
};

/// Descriptor for `SetAppScopesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setAppScopesRequestDescriptor = $convert.base64Decode(
    'ChNTZXRBcHBTY29wZXNSZXF1ZXN0Eg4KAmlkGAEgASgJUgJpZBIWCgZzY29wZXMYAiADKAlSBn'
    'Njb3Blcw==');
