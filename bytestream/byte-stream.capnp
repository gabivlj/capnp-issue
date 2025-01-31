@0x8f5d14e1c273738e;

interface Service {
    get @0 () -> (bsr :ByteStreamReturner);
}

interface ByteStreamReturner {
    getConnector @0 () -> (conn :Connector);
    inflighter @1 ();
}

interface Connector {
    connect @0 (down: ByteStream) -> (up :ByteStream);
}

interface ByteStream {
  write @0 (bytes :Data) -> stream;
}

using Go = import "/go.capnp";
$Go.package("bytestream");
$Go.import("bytestream");
