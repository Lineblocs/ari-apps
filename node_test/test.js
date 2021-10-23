const grpc = require("@grpc/grpc-js");
var protoLoader = require("@grpc/proto-loader");
const PROTO_PATH = "../lineblocs.proto";

const options = {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
};

var packageDefinition = protoLoader.loadSync(PROTO_PATH, options);
var loadedPkg = grpc.loadPackageDefinition(packageDefinition);
const LineblocsService = loadedPkg.grpc.Lineblocs;

const client = new LineblocsService(
  "localhost:9000",
  grpc.credentials.createInsecure()
);

setInterval( function() {
    // your methods:
    client.createBridge({
        hangup: false
    }, (error, resp) => {
            if (error) throw error;
            console.log("Successfully created bridge...");
  });
}, 1000);