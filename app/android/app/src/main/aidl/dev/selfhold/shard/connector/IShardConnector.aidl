// The Shard Connector API on Android (docs/CONNECTOR.md). Apps copy
// this file and IConnectorWatcher.aidl as is, in the same package
// (dev.selfhold.shard.connector), and bind with an explicit intent:
// action "dev.selfhold.shard.connector.BIND", package "dev.selfhold.shard".
//
// Requests and replies are serialized messages of proto/connector.proto.
// Shard knows the caller from Binder (package + signing certificate):
// there is no token on Android.
package dev.selfhold.shard.connector;

import dev.selfhold.shard.connector.IConnectorWatcher;

interface IShardConnector {
    // Version of this binder interface: 1.
    int version();

    // One unary method of the Connector service ("Hello", "RequestPairing",
    // "AwaitPairing", "Get", "List", "Put", "Delete", "AppKey", "Unpair"):
    // request is the serialized request message, the result a serialized
    // CallReply (code 0 = OK, payload = the reply message). It may block
    // (AwaitPairing up to 60 s): never call it on the main thread.
    byte[] call(String method, in byte[] request);

    // Starts a Watch: watcher gets serialized Change messages, then onEnd.
    // Returns a handle for unwatch.
    long watch(IConnectorWatcher watcher);
    void unwatch(long handle);
}
