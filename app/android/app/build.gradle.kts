import java.util.Properties

plugins {
    id("com.android.application")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

// Release signing: android/key.properties (storeFile, storePassword, keyAlias,
// keyPassword) or the same values in SHARD_KEYSTORE / SHARD_KEYSTORE_PASSWORD /
// SHARD_KEY_ALIAS / SHARD_KEY_PASSWORD (CI). Without them release builds use the debug key.
val keyProps = Properties().apply {
    val f = rootProject.file("key.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
fun signing(prop: String, env: String): String? = keyProps.getProperty(prop) ?: System.getenv(env)
val releaseStore = signing("storeFile", "SHARD_KEYSTORE")

android {
    namespace = "dev.selfhold.shard"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_17.toString()
    }

    // IShardConnector (ConnectorService, the Connector API for other apps).
    buildFeatures {
        aidl = true
    }

    defaultConfig {
        applicationId = "dev.selfhold.shard"
        // openProxyFileDescriptor (streamed reads in the documents provider).
        minSdk = 26
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    signingConfigs {
        if (releaseStore != null) {
            create("release") {
                storeFile = file(releaseStore)
                storePassword = signing("storePassword", "SHARD_KEYSTORE_PASSWORD")
                keyAlias = signing("keyAlias", "SHARD_KEY_ALIAS")
                keyPassword = signing("keyPassword", "SHARD_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.getByName(if (releaseStore != null) "release" else "debug")
        }
    }
}

flutter {
    source = "../.."
}

dependencies {
    implementation(files("libs/shard.aar"))
    // Unlock by fingerprint (Biometric.kt); AppCompat for its dialog on Android 8.
    implementation("androidx.biometric:biometric:1.1.0")
    implementation("androidx.appcompat:appcompat:1.7.0")
}
