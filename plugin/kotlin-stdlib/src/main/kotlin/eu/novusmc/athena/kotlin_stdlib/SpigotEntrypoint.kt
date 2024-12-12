package eu.novusmc.athena.kotlin_stdlib

import org.bukkit.plugin.java.JavaPlugin
import org.bukkit.plugin.java.annotation.plugin.ApiVersion
import org.bukkit.plugin.java.annotation.plugin.Plugin

@Plugin(name = "kotlin-stdlib", version = "2.1.0")
@ApiVersion(ApiVersion.Target.v1_20)
class SpigotEntrypoint : JavaPlugin()
