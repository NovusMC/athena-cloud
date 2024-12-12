dependencies {
    implementation(project(":common", "shadow"))
    implementation("de.pauhull.novus-utils", "common", "0.5.2")
    compileOnly("com.velocitypowered", "velocity-api", "3.2.0-SNAPSHOT")
    kapt("com.velocitypowered", "velocity-api", "3.2.0-SNAPSHOT")
}
