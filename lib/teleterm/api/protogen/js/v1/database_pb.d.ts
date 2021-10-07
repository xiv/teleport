// package: teleport.terminal.v1
// file: v1/database.proto

import * as jspb from "google-protobuf";
import * as v1_label_pb from "../v1/label_pb";

export class Database extends jspb.Message {
  getUri(): string;
  setUri(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDesc(): string;
  setDesc(value: string): void;

  getProtocol(): string;
  setProtocol(value: string): void;

  getType(): string;
  setType(value: string): void;

  getHostname(): string;
  setHostname(value: string): void;

  getAddr(): string;
  setAddr(value: string): void;

  clearLabelsList(): void;
  getLabelsList(): Array<v1_label_pb.Label>;
  setLabelsList(value: Array<v1_label_pb.Label>): void;
  addLabels(value?: v1_label_pb.Label, index?: number): v1_label_pb.Label;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Database.AsObject;
  static toObject(includeInstance: boolean, msg: Database): Database.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Database, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Database;
  static deserializeBinaryFromReader(message: Database, reader: jspb.BinaryReader): Database;
}

export namespace Database {
  export type AsObject = {
    uri: string,
    name: string,
    desc: string,
    protocol: string,
    type: string,
    hostname: string,
    addr: string,
    labelsList: Array<v1_label_pb.Label.AsObject>,
  }
}

