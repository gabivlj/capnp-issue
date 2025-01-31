use capnp;
use capnpc;

fn main() -> capnp::Result<()> {
    println!("cargo:rerun-if-changed=./src/");
    capnpc::CompilerCommand::new()
        .src_prefix("src/")
        .file("src/byte-stream.capnp")
        .output_path("src/")
        .run()
}
