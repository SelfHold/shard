// Gets the changes of one Watch (see IShardConnector.aidl).
package dev.selfhold.shard.connector;

oneway interface IConnectorWatcher {
    // A serialized Change message.
    void onChange(in byte[] change);
    // The watch ended: a serialized CallReply with the reason.
    void onEnd(in byte[] reply);
}
