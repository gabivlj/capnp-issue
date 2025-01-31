repro steps:

- you need Go installed
- you need Rust installed

Build the server:
```
make build
```


Build rust client:
```
cd rs-client && cargo build --release
```

Repro in background:
```
bin/repro
```

Another terminal:
```
cd rs-client && cargo build --release && ./target/release/cc-test ../repro.sock
```
