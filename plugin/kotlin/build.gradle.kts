import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    kotlin("jvm") version "2.1.21"
    kotlin("plugin.serialization") version "2.1.21"
    application
    id("com.google.protobuf") version "0.9.5"
}

group = "com.canopy.plugin"
version = "1.0.0"

repositories {
    mavenCentral()
    // Consensys/Teku Cloudsmith repository for BLS library
    maven {
        url = uri("https://dl.cloudsmith.io/public/consensys/teku/maven/")
    }
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

    // Logging
    implementation("io.github.microutils:kotlin-logging-jvm:3.0.5")
    implementation("ch.qos.logback:logback-classic:1.4.14")

    // JSON handling (for config file loading)
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.2")

    // Unix domain socket support
    implementation("com.kohlschutter.junixsocket:junixsocket-core:2.9.0")

    // HTTP client for RPC calls
    implementation("com.squareup.okhttp3:okhttp:4.12.0")

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
    mainClass.set("com.canopy.plugin.MainKt")
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

tasks.register<JavaExec>("dev") {
    group = "application"
    description = "Run the plugin in development mode"
    mainClass.set("com.canopy.plugin.MainKt")
    classpath = sourceSets["main"].runtimeClasspath
    jvmArgs = listOf("-Xmx512m")
}

tasks.register("typeCheck") {
    group = "verification"
    description = "Type check Kotlin code"
    dependsOn("compileKotlin")
}

tasks.register("validate") {
    group = "verification"
    description = "Run all validation checks"
    dependsOn("typeCheck", "test")
}

// Fat JAR task for standalone execution
tasks.register<Jar>("fatJar") {
    group = "build"
    description = "Build a fat JAR with all dependencies"
    archiveClassifier.set("all")
    duplicatesStrategy = DuplicatesStrategy.EXCLUDE
    manifest {
        attributes["Main-Class"] = "com.canopy.plugin.MainKt"
    }
    from(configurations.runtimeClasspath.get().map { if (it.isDirectory) it else zipTree(it) })
    with(tasks.jar.get())
}
