use capnp_rpc::{rpc_twoparty_capnp, twoparty, RpcSystem};
use futures::AsyncReadExt;

use capnp::capability::{FromClientHook, Promise};
mod byte_stream_capnp;

pub mod byte_stream {
    include!("byte_stream_capnp.rs");
}

pub struct GetFunction;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Hello, world!");
    let args: Vec<String> = ::std::env::args().collect();
    if args.len() != 2 {
        println!("usage: {} UNIX_SOCKET", args[0]);
        return Ok(());
    }

    let addr = args[1].clone();
    let _ = tokio::task::LocalSet::new()
        .run_until(async move {
            let stream = tokio::net::UnixStream::connect(&addr).await.unwrap();
            let (reader, writer) =
                tokio_util::compat::TokioAsyncReadCompatExt::compat(stream).split();

            let rpc_network = Box::new(twoparty::VatNetwork::new(
                futures::io::BufReader::new(reader),
                futures::io::BufWriter::new(writer),
                rpc_twoparty_capnp::Side::Client,
                Default::default(),
            ));
            let mut rpc_system = RpcSystem::new(rpc_network, None);
            let service: byte_stream::service::Client =
                rpc_system.bootstrap(rpc_twoparty_capnp::Side::Server);
            tokio::task::spawn_local(rpc_system);

            let request = service.get_request();
            let res = request.send();
            let bs_get = res.promise.await.expect("i dont care about failures");
            let bsr = bs_get.get().unwrap().get_bsr().unwrap();

            // Start some inflight promise that we will await later,
            // unrelated to the connect call
            let inflighter_too = bsr.inflighter_request().send().promise;
            // Start the pipeline
            let promise = bsr.get_connector_request().send().pipeline;
            // Get connector
            let connector = promise.get_conn().connect_request();
            // Send the connector and get the pipeline
            let bs = connector.send().pipeline;
            // Get up bytestream
            let up = bs.get_up();
            // Get the write request
            let mut res = up.write_request();
            // Set the bytes
            let mut res_req = res.get();
            res_req.set_bytes("hello world".as_bytes());
            // Send everything now
            let _ = res.send().await.unwrap();
            // Wait for inflighter
            println!("Whole thing finished");
            inflighter_too.await.unwrap();
            // Bye
            Ok::<(), ()>(())
        })
        .await;

    Ok(())
}
