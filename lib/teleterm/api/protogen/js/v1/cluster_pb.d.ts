// package: teleport.terminal.v1
// file: v1/cluster.proto

import * as jspb from "google-protobuf";

export class Cluster extends jspb.Message {
  getUri(): string;
  setUri(value: string): void;

  getName(): string;
  setName(value: string): void;

  getConnected(): boolean;
  setConnected(value: boolean): void;

  hasAcl(): boolean;
  clearAcl(): void;
  getAcl(): ClusterACL | undefined;
  setAcl(value?: ClusterACL): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Cluster.AsObject;
  static toObject(includeInstance: boolean, msg: Cluster): Cluster.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Cluster, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Cluster;
  static deserializeBinaryFromReader(message: Cluster, reader: jspb.BinaryReader): Cluster;
}

export namespace Cluster {
  export type AsObject = {
    uri: string,
    name: string,
    connected: boolean,
    acl?: ClusterACL.AsObject,
  }
}

export class ResourceAccess extends jspb.Message {
  getList(): boolean;
  setList(value: boolean): void;

  getRead(): boolean;
  setRead(value: boolean): void;

  getEdit(): boolean;
  setEdit(value: boolean): void;

  getCreate(): boolean;
  setCreate(value: boolean): void;

  getDelete(): boolean;
  setDelete(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceAccess.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceAccess): ResourceAccess.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ResourceAccess, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceAccess;
  static deserializeBinaryFromReader(message: ResourceAccess, reader: jspb.BinaryReader): ResourceAccess;
}

export namespace ResourceAccess {
  export type AsObject = {
    list: boolean,
    read: boolean,
    edit: boolean,
    create: boolean,
    pb_delete: boolean,
  }
}

export class ClusterACL extends jspb.Message {
  hasSessions(): boolean;
  clearSessions(): void;
  getSessions(): ResourceAccess | undefined;
  setSessions(value?: ResourceAccess): void;

  hasAuthConnectors(): boolean;
  clearAuthConnectors(): void;
  getAuthConnectors(): ResourceAccess | undefined;
  setAuthConnectors(value?: ResourceAccess): void;

  hasRoles(): boolean;
  clearRoles(): void;
  getRoles(): ResourceAccess | undefined;
  setRoles(value?: ResourceAccess): void;

  hasUsers(): boolean;
  clearUsers(): void;
  getUsers(): ResourceAccess | undefined;
  setUsers(value?: ResourceAccess): void;

  hasTrustedClusters(): boolean;
  clearTrustedClusters(): void;
  getTrustedClusters(): ResourceAccess | undefined;
  setTrustedClusters(value?: ResourceAccess): void;

  hasEvents(): boolean;
  clearEvents(): void;
  getEvents(): ResourceAccess | undefined;
  setEvents(value?: ResourceAccess): void;

  hasTokens(): boolean;
  clearTokens(): void;
  getTokens(): ResourceAccess | undefined;
  setTokens(value?: ResourceAccess): void;

  hasServers(): boolean;
  clearServers(): void;
  getServers(): ResourceAccess | undefined;
  setServers(value?: ResourceAccess): void;

  hasApps(): boolean;
  clearApps(): void;
  getApps(): ResourceAccess | undefined;
  setApps(value?: ResourceAccess): void;

  hasDbs(): boolean;
  clearDbs(): void;
  getDbs(): ResourceAccess | undefined;
  setDbs(value?: ResourceAccess): void;

  hasKubeservers(): boolean;
  clearKubeservers(): void;
  getKubeservers(): ResourceAccess | undefined;
  setKubeservers(value?: ResourceAccess): void;

  clearSshLoginsList(): void;
  getSshLoginsList(): Array<string>;
  setSshLoginsList(value: Array<string>): void;
  addSshLogins(value: string, index?: number): string;

  hasAccessRequests(): boolean;
  clearAccessRequests(): void;
  getAccessRequests(): ResourceAccess | undefined;
  setAccessRequests(value?: ResourceAccess): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClusterACL.AsObject;
  static toObject(includeInstance: boolean, msg: ClusterACL): ClusterACL.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClusterACL, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClusterACL;
  static deserializeBinaryFromReader(message: ClusterACL, reader: jspb.BinaryReader): ClusterACL;
}

export namespace ClusterACL {
  export type AsObject = {
    sessions?: ResourceAccess.AsObject,
    authConnectors?: ResourceAccess.AsObject,
    roles?: ResourceAccess.AsObject,
    users?: ResourceAccess.AsObject,
    trustedClusters?: ResourceAccess.AsObject,
    events?: ResourceAccess.AsObject,
    tokens?: ResourceAccess.AsObject,
    servers?: ResourceAccess.AsObject,
    apps?: ResourceAccess.AsObject,
    dbs?: ResourceAccess.AsObject,
    kubeservers?: ResourceAccess.AsObject,
    sshLoginsList: Array<string>,
    accessRequests?: ResourceAccess.AsObject,
  }
}

export class ClusterAuthSettings extends jspb.Message {
  getType(): string;
  setType(value: string): void;

  getSecondFactor(): string;
  setSecondFactor(value: string): void;

  hasU2f(): boolean;
  clearU2f(): void;
  getU2f(): AuthSettingsU2F | undefined;
  setU2f(value?: AuthSettingsU2F): void;

  clearAuthProvidersList(): void;
  getAuthProvidersList(): Array<AuthProvider>;
  setAuthProvidersList(value: Array<AuthProvider>): void;
  addAuthProviders(value?: AuthProvider, index?: number): AuthProvider;

  getHasMessageOfTheDay(): boolean;
  setHasMessageOfTheDay(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClusterAuthSettings.AsObject;
  static toObject(includeInstance: boolean, msg: ClusterAuthSettings): ClusterAuthSettings.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClusterAuthSettings, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClusterAuthSettings;
  static deserializeBinaryFromReader(message: ClusterAuthSettings, reader: jspb.BinaryReader): ClusterAuthSettings;
}

export namespace ClusterAuthSettings {
  export type AsObject = {
    type: string,
    secondFactor: string,
    u2f?: AuthSettingsU2F.AsObject,
    authProvidersList: Array<AuthProvider.AsObject>,
    hasMessageOfTheDay: boolean,
  }
}

export class AuthProvider extends jspb.Message {
  getType(): string;
  setType(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDisplay(): string;
  setDisplay(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AuthProvider.AsObject;
  static toObject(includeInstance: boolean, msg: AuthProvider): AuthProvider.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AuthProvider, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AuthProvider;
  static deserializeBinaryFromReader(message: AuthProvider, reader: jspb.BinaryReader): AuthProvider;
}

export namespace AuthProvider {
  export type AsObject = {
    type: string,
    name: string,
    display: string,
  }
}

export class AuthSettingsU2F extends jspb.Message {
  getAppId(): string;
  setAppId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AuthSettingsU2F.AsObject;
  static toObject(includeInstance: boolean, msg: AuthSettingsU2F): AuthSettingsU2F.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AuthSettingsU2F, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AuthSettingsU2F;
  static deserializeBinaryFromReader(message: AuthSettingsU2F, reader: jspb.BinaryReader): AuthSettingsU2F;
}

export namespace AuthSettingsU2F {
  export type AsObject = {
    appId: string,
  }
}

