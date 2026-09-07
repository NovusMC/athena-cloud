dependencies {
    api(kotlin("stdlib", "2.4.20"))
    compileOnly("io.papermc.paper", "paper-api", "1.21.4-R0.1-SNAPSHOT")
    compileOnly("org.spigotmc", "plugin-annotations", "1.2.3-SNAPSHOT")
    compileOnly("com.velocitypowered", "velocity-api", "3.2.0-SNAPSHOT")
    kapt("com.velocitypowered", "velocity-api", "3.2.0-SNAPSHOT")
    kapt("org.spigotmc", "plugin-annotations", "1.2.3-SNAPSHOT")
}
