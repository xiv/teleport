// package: teleport.terminal.v1
// file: v1/service.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as v1_cluster_pb from "../v1/cluster_pb";
import * as v1_cluster_login_challenge_pb from "../v1/cluster_login_challenge_pb";
import * as v1_database_pb from "../v1/database_pb";
import * as v1_gateway_pb from "../v1/gateway_pb";
import * as v1_server_pb from "../v1/server_pb";

export class CreateClusterRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateClusterRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateClusterRequest): CreateClusterRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateClusterRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateClusterRequest;
  static deserializeBinaryFromReader(message: CreateClusterRequest, reader: jspb.BinaryReader): CreateClusterRequest;
}

export namespace CreateClusterRequest {
  export type AsObject = {
    name: string,
  }
}

export class ListClustersRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListClustersRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListClustersRequest): ListClustersRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListClustersRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListClustersRequest;
  static deserializeBinaryFromReader(message: ListClustersRequest, reader: jspb.BinaryReader): ListClustersRequest;
}

export namespace ListClustersRequest {
  export type AsObject = {
  }
}

export class ListClustersResponse extends jspb.Message {
  clearClustersList(): void;
  getClustersList(): Array<v1_cluster_pb.Cluster>;
  setClustersList(value: Array<v1_cluster_pb.Cluster>): void;
  addClusters(value?: v1_cluster_pb.Cluster, index?: number): v1_cluster_pb.Cluster;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListClustersResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListClustersResponse): ListClustersResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListClustersResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListClustersResponse;
  static deserializeBinaryFromReader(message: ListClustersResponse, reader: jspb.BinaryReader): ListClustersResponse;
}

export namespace ListClustersResponse {
  export type AsObject = {
    clustersList: Array<v1_cluster_pb.Cluster.AsObject>,
  }
}

export class CreateClusterLoginChallengeRequest extends jspb.Message {
  getClusterId(): string;
  setClusterId(value: string): void;

  getLogin(): string;
  setLogin(value: string): void;

  getPassword(): string;
  setPassword(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateClusterLoginChallengeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateClusterLoginChallengeRequest): CreateClusterLoginChallengeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateClusterLoginChallengeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateClusterLoginChallengeRequest;
  static deserializeBinaryFromReader(message: CreateClusterLoginChallengeRequest, reader: jspb.BinaryReader): CreateClusterLoginChallengeRequest;
}

export namespace CreateClusterLoginChallengeRequest {
  export type AsObject = {
    clusterId: string,
    login: string,
    password: string,
  }
}

export class SolveClusterLoginChallengeRequest extends jspb.Message {
  getClusterId(): string;
  setClusterId(value: string): void;

  getChallengeId(): string;
  setChallengeId(value: string): void;

  hasSolvedChallenge(): boolean;
  clearSolvedChallenge(): void;
  getSolvedChallenge(): v1_cluster_login_challenge_pb.SolvedClusterLoginChallenge | undefined;
  setSolvedChallenge(value?: v1_cluster_login_challenge_pb.SolvedClusterLoginChallenge): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SolveClusterLoginChallengeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SolveClusterLoginChallengeRequest): SolveClusterLoginChallengeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SolveClusterLoginChallengeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SolveClusterLoginChallengeRequest;
  static deserializeBinaryFromReader(message: SolveClusterLoginChallengeRequest, reader: jspb.BinaryReader): SolveClusterLoginChallengeRequest;
}

export namespace SolveClusterLoginChallengeRequest {
  export type AsObject = {
    clusterId: string,
    challengeId: string,
    solvedChallenge?: v1_cluster_login_challenge_pb.SolvedClusterLoginChallenge.AsObject,
  }
}

export class SolveClusterLoginChallengeResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SolveClusterLoginChallengeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SolveClusterLoginChallengeResponse): SolveClusterLoginChallengeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SolveClusterLoginChallengeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SolveClusterLoginChallengeResponse;
  static deserializeBinaryFromReader(message: SolveClusterLoginChallengeResponse, reader: jspb.BinaryReader): SolveClusterLoginChallengeResponse;
}

export namespace SolveClusterLoginChallengeResponse {
  export type AsObject = {
  }
}

export class ListDatabasesRequest extends jspb.Message {
  getClusterUri(): string;
  setClusterUri(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDatabasesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListDatabasesRequest): ListDatabasesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListDatabasesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDatabasesRequest;
  static deserializeBinaryFromReader(message: ListDatabasesRequest, reader: jspb.BinaryReader): ListDatabasesRequest;
}

export namespace ListDatabasesRequest {
  export type AsObject = {
    clusterUri: string,
  }
}

export class ListDatabasesResponse extends jspb.Message {
  clearDatabasesList(): void;
  getDatabasesList(): Array<v1_database_pb.Database>;
  setDatabasesList(value: Array<v1_database_pb.Database>): void;
  addDatabases(value?: v1_database_pb.Database, index?: number): v1_database_pb.Database;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDatabasesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListDatabasesResponse): ListDatabasesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListDatabasesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDatabasesResponse;
  static deserializeBinaryFromReader(message: ListDatabasesResponse, reader: jspb.BinaryReader): ListDatabasesResponse;
}

export namespace ListDatabasesResponse {
  export type AsObject = {
    databasesList: Array<v1_database_pb.Database.AsObject>,
  }
}

export class CreateGatewayRequest extends jspb.Message {
  getTargetUri(): string;
  setTargetUri(value: string): void;

  getPort(): string;
  setPort(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateGatewayRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateGatewayRequest): CreateGatewayRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateGatewayRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateGatewayRequest;
  static deserializeBinaryFromReader(message: CreateGatewayRequest, reader: jspb.BinaryReader): CreateGatewayRequest;
}

export namespace CreateGatewayRequest {
  export type AsObject = {
    targetUri: string,
    port: string,
  }
}

export class ListGatewaysRequest extends jspb.Message {
  clearClusterIdsList(): void;
  getClusterIdsList(): Array<string>;
  setClusterIdsList(value: Array<string>): void;
  addClusterIds(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListGatewaysRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListGatewaysRequest): ListGatewaysRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListGatewaysRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListGatewaysRequest;
  static deserializeBinaryFromReader(message: ListGatewaysRequest, reader: jspb.BinaryReader): ListGatewaysRequest;
}

export namespace ListGatewaysRequest {
  export type AsObject = {
    clusterIdsList: Array<string>,
  }
}

export class ListGatewaysResponse extends jspb.Message {
  clearGatewaysList(): void;
  getGatewaysList(): Array<v1_gateway_pb.Gateway>;
  setGatewaysList(value: Array<v1_gateway_pb.Gateway>): void;
  addGateways(value?: v1_gateway_pb.Gateway, index?: number): v1_gateway_pb.Gateway;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListGatewaysResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListGatewaysResponse): ListGatewaysResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListGatewaysResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListGatewaysResponse;
  static deserializeBinaryFromReader(message: ListGatewaysResponse, reader: jspb.BinaryReader): ListGatewaysResponse;
}

export namespace ListGatewaysResponse {
  export type AsObject = {
    gatewaysList: Array<v1_gateway_pb.Gateway.AsObject>,
  }
}

export class DeleteGatewayRequest extends jspb.Message {
  getGatewayUri(): string;
  setGatewayUri(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteGatewayRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteGatewayRequest): DeleteGatewayRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteGatewayRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteGatewayRequest;
  static deserializeBinaryFromReader(message: DeleteGatewayRequest, reader: jspb.BinaryReader): DeleteGatewayRequest;
}

export namespace DeleteGatewayRequest {
  export type AsObject = {
    gatewayUri: string,
  }
}

export class ListServersRequest extends jspb.Message {
  getClusterUri(): string;
  setClusterUri(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListServersRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListServersRequest): ListServersRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListServersRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListServersRequest;
  static deserializeBinaryFromReader(message: ListServersRequest, reader: jspb.BinaryReader): ListServersRequest;
}

export namespace ListServersRequest {
  export type AsObject = {
    clusterUri: string,
  }
}

export class ListServersResponse extends jspb.Message {
  clearServersList(): void;
  getServersList(): Array<v1_server_pb.Server>;
  setServersList(value: Array<v1_server_pb.Server>): void;
  addServers(value?: v1_server_pb.Server, index?: number): v1_server_pb.Server;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListServersResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListServersResponse): ListServersResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListServersResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListServersResponse;
  static deserializeBinaryFromReader(message: ListServersResponse, reader: jspb.BinaryReader): ListServersResponse;
}

export namespace ListServersResponse {
  export type AsObject = {
    serversList: Array<v1_server_pb.Server.AsObject>,
  }
}

export class GetClusterAuthSettingsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetClusterAuthSettingsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetClusterAuthSettingsRequest): GetClusterAuthSettingsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetClusterAuthSettingsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetClusterAuthSettingsRequest;
  static deserializeBinaryFromReader(message: GetClusterAuthSettingsRequest, reader: jspb.BinaryReader): GetClusterAuthSettingsRequest;
}

export namespace GetClusterAuthSettingsRequest {
  export type AsObject = {
    name: string,
  }
}

export class EmptyResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EmptyResponse.AsObject;
  static toObject(includeInstance: boolean, msg: EmptyResponse): EmptyResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: EmptyResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EmptyResponse;
  static deserializeBinaryFromReader(message: EmptyResponse, reader: jspb.BinaryReader): EmptyResponse;
}

export namespace EmptyResponse {
  export type AsObject = {
  }
}

