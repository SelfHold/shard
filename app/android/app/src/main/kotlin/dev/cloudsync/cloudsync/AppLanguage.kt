package dev.cloudsync.cloudsync

import android.content.Context
import android.content.res.Configuration
import org.json.JSONObject
import java.io.File
import java.util.Locale

/**
 * The UI language the user picked in the app, never the system locale:
 * English unless Russian was chosen in Settings. The Flutter side keeps the
 * choice in `<filesDir>/app.json` (`AppSettings`, per device, never synced).
 * Strings shown from Kotlin (notifications, the file picker root) are read
 * through [localized], so `values-ru/` is used only when the user chose it.
 */
object AppLanguage {
    private val supported = setOf("en", "ru")

    fun code(context: Context): String =
        runCatching {
            File(context.filesDir, "app.json").takeIf { it.exists() }?.readText()?.let { JSONObject(it).optString("language") }
        }.getOrNull()?.takeIf { it in supported } ?: "en"

    /** [context] with its resources in the chosen language. */
    fun localized(context: Context): Context {
        val config = Configuration(context.resources.configuration)
        config.setLocale(Locale.forLanguageTag(code(context)))
        return context.createConfigurationContext(config)
    }
}
