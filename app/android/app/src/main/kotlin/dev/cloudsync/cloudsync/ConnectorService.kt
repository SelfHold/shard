package dev.cloudsync.cloudsync

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Binder
import android.os.Build
import android.os.IBinder
import dev.cloudsync.connector.ICloudSyncConnector
import dev.cloudsync.connector.IConnectorWatcher
import dev.cloudsync.mobilebind.ConnectorSink
import dev.cloudsync.mobilebind.Mobilebind
import java.security.MessageDigest

/**
 * The Connector API for other apps (docs/CONNECTOR.md): an exported bound
 * service. Anyone may bind; what a caller may do is decided by the core
 * from the pairing the user approved for its package and signing
 * certificate, which are read from Binder here, never taken from the
 * caller.
 */
class ConnectorService : Service() {
    override fun onBind(intent: Intent?): IBinder {
        Core.start(this)
        return binder
    }

    private val binder = object : ICloudSyncConnector.Stub() {
        override fun version(): Int = 1

        override fun call(method: String, request: ByteArray?): ByteArray {
            val who = caller() ?: return refused()
            return Mobilebind.connectorCall(who.first, who.second, method, request ?: ByteArray(0))
        }

        override fun watch(watcher: IConnectorWatcher): Long {
            val who = caller()
            if (who == null) {
                runCatching { watcher.onEnd(refused()) }
                return 0
            }
            val id = Mobilebind.connectorWatch(who.first, who.second, object : ConnectorSink {
                override fun onChange(change: ByteArray?) {
                    watcher.onChange(change ?: ByteArray(0)) // throws when the client is gone
                }

                override fun onEnd(reply: ByteArray?) {
                    runCatching { watcher.onEnd(reply ?: ByteArray(0)) }
                }
            })
            // The client's process died: stop its watch.
            runCatching { watcher.asBinder().linkToDeath({ Mobilebind.connectorUnwatch(id) }, 0) }
            return id
        }

        override fun unwatch(handle: Long) = Mobilebind.connectorUnwatch(handle)
    }

    /** The calling package and the SHA-256 of its signing certificate(s). */
    private fun caller(): Pair<String, String>? {
        val uid = Binder.getCallingUid()
        val pkgs = packageManager.getPackagesForUid(uid) ?: return null
        // A shared user id would make the package ambiguous.
        val pkg = pkgs.singleOrNull() ?: return null
        val certs = signingCerts(pkg) ?: return null
        val md = MessageDigest.getInstance("SHA-256")
        val digest = certs.map { c -> md.digest(c).joinToString("") { "%02x".format(it) } }.sorted().joinToString(":")
        return pkg to digest
    }

    @Suppress("DEPRECATION")
    private fun signingCerts(pkg: String): List<ByteArray>? = runCatching {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            val info = packageManager.getPackageInfo(pkg, PackageManager.GET_SIGNING_CERTIFICATES).signingInfo
                ?: return null
            info.apkContentsSigners.map { it.toByteArray() }
        } else {
            packageManager.getPackageInfo(pkg, PackageManager.GET_SIGNATURES).signatures?.map { it.toByteArray() }
        }
    }.getOrNull()?.takeIf { it.isNotEmpty() }

    private fun refused(): ByteArray = Mobilebind.connectorRefused("the calling package could not be identified")

    companion object {
        private const val CHANNEL = "apps"
        private const val NOTICE_ID = 2

        /**
         * Shows (or removes) the notice that an app waits for the user's
         * answer; called on core events (Core.kt).
         */
        fun updateNotice(context: Context) {
            val nm = context.getSystemService(NotificationManager::class.java)
            val n = Mobilebind.pendingPairings()
            if (n <= 0L) {
                nm.cancel(NOTICE_ID)
                return
            }
            nm.createNotificationChannel(
                NotificationChannel(CHANNEL, context.getString(R.string.apps_channel_name), NotificationManager.IMPORTANCE_HIGH).apply {
                    description = context.getString(R.string.apps_channel_description)
                },
            )
            val open = PendingIntent.getActivity(
                context, 1, Intent(context, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
            )
            // A request for secrets (passwords, SSH keys, servers) says so
            // plainly, one line per scope.
            val asks = listOf(
                "passwords" to R.string.apps_notice_passwords_title,
                "ssh_keys" to R.string.apps_notice_ssh_keys_title,
                "servers" to R.string.apps_notice_servers_title,
            ).mapNotNull { (scope, res) ->
                Mobilebind.scopePairing(scope).takeIf { it.isNotEmpty() }?.let { context.getString(res, it) }
            }
            val title = asks.firstOrNull() ?: context.getString(R.string.apps_notice_title)
            val text = (asks.drop(1) + context.getString(R.string.apps_notice_text)).joinToString("\n")
            nm.notify(
                NOTICE_ID,
                Notification.Builder(context, CHANNEL)
                    .setContentTitle(title)
                    .setContentText(text)
                    .setStyle(Notification.BigTextStyle().bigText(text))
                    .setSmallIcon(R.drawable.ic_stat_sync)
                    .setContentIntent(open)
                    .setAutoCancel(true)
                    .build(),
            )
        }
    }
}
