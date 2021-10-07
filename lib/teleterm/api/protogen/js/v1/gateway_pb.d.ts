// package: teleport.terminal.v1
// file: v1/gateway.proto

import * as jspb from "google-protobuf";

export class Gateway extends jspb.Message {
  getUri(): string;
  setUri(value: string): void;

  getResourceName(): string;
  setResourceName(value: string): void;

  getLocalAddress(): string;
  setLocalAddress(value: string): void;

  getLocalPort(): string;
  setLocalPort(value: string): void;

  getProtocol(): string;
  setProtocol(value: string): void;

  getHostId(): string;
  setHostId(value: string): void;

  getClusterId(): string;
  setClusterId(value: string): void;

  getStatus(): Gateway.GatewayStatusMap[keyof Gateway.GatewayStatusMap];
  setStatus(value: Gateway.GatewayStatusMap[keyof Gateway.GatewayStatusMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Gateway.AsObject;
  static toObject(includeInstance: boolean, msg: Gateway): Gateway.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Gateway, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Gateway;
  static deserializeBinaryFromReader(message: Gateway, reader: jspb.BinaryReader): Gateway;
}

export namespace Gateway {
  export type AsObject = {
    uri: string,
    resourceName: string,
    localAddress: string,
    localPort: string,
    protocol: string,
    hostId: string,
    clusterId: string,
    status: Gateway.GatewayStatusMap[keyof Gateway.GatewayStatusMap],
  }

  export interface GatewayStatusMap {
    CONNECTED: 0;
    DISCONNECTED: 1;
  }

  export const GatewayStatus: GatewayStatusMap;
}

