import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    kotlin("jvm") version "1.9.22"
    application
    id("com.google.protobuf") version "0.9.4"
    // Removed ktlint due to issues with generated protobuf files
}

group = "com.canopy.plugin"
version = "1.0.0"

repositories {
    mavenCentral()
}

dependencies {
    // Kotlin standard library
    implementation(kotlin("stdlib"))
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.0")

    // Protobuf
    implementation("com.google.protobuf:protobuf-java:3.25.2")
    implementation("com.google.protobuf:protobuf-kotlin:3.25.2")

    // Networking
    implementation("io.ktor:ktor-network:2.3.7")

    // Logging
    implementation("io.github.microutils:kotlin-logging-jvm:3.0.5")
    implementation("ch.qos.logback:logback-classic:1.4.14")

    // JSON handling
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.2")

    // Testing
    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.1")
    testImplementation("io.mockk:mockk:1.13.9")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.8.0")
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
    kotlinOptions {
        jvmTarget = "11"
        freeCompilerArgs = listOf("-Xjsr305=strict")
    }
}

tasks.withType<JavaCompile> {
    targetCompatibility = "11"
    sourceCompatibility = "11"
}

tasks.test {
    useJUnitPlatform()
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
