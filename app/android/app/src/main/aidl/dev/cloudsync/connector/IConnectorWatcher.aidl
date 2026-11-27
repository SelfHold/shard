// Gets the changes of one Watch (see ICloudSyncConnector.aidl).
package dev.cloudsync.connector;

oneway interface IConnectorWatcher {
    // A serialized Change message.
    void onChange(in byte[] change);
    // The watch ended: a serialized CallReply with the reason.
    void onEnd(in byte[] reply);
}
