dependencies {
    api(project(":common", "shadow"))
    implementation("de.pauhull.novus-utils", "common", "0.5.2")
    compileOnly("io.papermc.paper", "paper-api", "1.21.4-R0.1-SNAPSHOT")
    compileOnly("org.spigotmc", "plugin-annotations", "1.2.3-SNAPSHOT")
    kapt("org.spigotmc", "plugin-annotations", "1.2.3-SNAPSHOT")
}

tasks {
    shadowJar {
        // paper includes an outdated protobuf version
        relocate("com.google.protobuf", "eu.novusmc.athena.shadow.com.google.protobuf")
    }
}
