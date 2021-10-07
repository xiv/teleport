// package: teleport.terminal.v1
// file: v1/server.proto

import * as jspb from "google-protobuf";
import * as v1_label_pb from "../v1/label_pb";

export class Server extends jspb.Message {
  getUri(): string;
  setUri(value: string): void;

  getTunnel(): boolean;
  setTunnel(value: boolean): void;

  getName(): string;
  setName(value: string): void;

  getClusterId(): string;
  setClusterId(value: string): void;

  getHostname(): string;
  setHostname(value: string): void;

  getAddr(): string;
  setAddr(value: string): void;

  clearLabelsList(): void;
  getLabelsList(): Array<v1_label_pb.Label>;
  setLabelsList(value: Array<v1_label_pb.Label>): void;
  addLabels(value?: v1_label_pb.Label, index?: number): v1_label_pb.Label;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Server.AsObject;
  static toObject(includeInstance: boolean, msg: Server): Server.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Server, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Server;
  static deserializeBinaryFromReader(message: Server, reader: jspb.BinaryReader): Server;
}

export namespace Server {
  export type AsObject = {
    uri: string,
    tunnel: boolean,
    name: string,
    clusterId: string,
    hostname: string,
    addr: string,
    labelsList: Array<v1_label_pb.Label.AsObject>,
  }
}

