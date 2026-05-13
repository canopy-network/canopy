import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    kotlin("jvm") version "2.1.21"
    kotlin("plugin.serialization") version "2.1.21"
    application
    id("com.google.protobuf") version "0.9.5"
}

group = "com.canopy.tutorial"
version = "1.0.0"

repositories {
    mavenCentral()
    // Consensys Maven for jblst native library
    maven {
        url = uri("https://artifacts.consensys.net/public/maven/maven/")
    }
}

// Use Java toolchain to ensure compatible JDK version
kotlin {
    jvmToolchain(21)
}

dependencies {
    // Kotlin standard library
    implementation(kotlin("stdlib"))

    // Protobuf
    implementation("com.google.protobuf:protobuf-java:3.25.2")
    implementation("com.google.protobuf:protobuf-kotlin:3.25.2")

    // JSON handling
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.2")

    // BLS12-381 cryptography (for transaction signing) - using jblst directly with custom DST
    implementation("tech.pegasys:jblst:0.3.12")

    // Testing
    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.1")
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:3.25.2"
    }
    generateProtoTasks {
        all().forEach { task ->
            task.builtins {
                create("kotlin")
            }
        }
    }
}

application {
    mainClass.set("com.canopy.tutorial.MainKt")
}

tasks.withType<KotlinCompile> {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_21)
        freeCompilerArgs.add("-Xjsr305=strict")
    }
}

tasks.test {
    useJUnitPlatform()
    testLogging {
        events("passed", "skipped", "failed", "standardOut", "standardError")
        showStandardStreams = true
        exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
    }
}
