package eu.novusmc.athena.paper

import com.google.protobuf.Message
import de.pauhull.novus_utils.common.PluginConfig
import eu.novusmc.athena.common.Configuration
import eu.novusmc.athena.common.Packet
import eu.novusmc.athena.common.Protocol
import java.io.File
import java.net.Socket
import org.bukkit.plugin.java.JavaPlugin
import org.bukkit.plugin.java.annotation.dependency.Dependency
import org.bukkit.plugin.java.annotation.plugin.ApiVersion
import org.bukkit.plugin.java.annotation.plugin.Plugin

@Plugin(
    name = "athena",
    version = "1.0.0", // x-release-please-version
)
@ApiVersion(ApiVersion.Target.v1_20)
@Dependency("kotlin-stdlib")
class AthenaPaperPlugin : JavaPlugin() {

    private var shuttingDown = false
    private var sock: Socket? = null

    override fun onEnable() {
        try {
            val cfg =
                PluginConfig.copyAndLoad(
                    Configuration::class.java,
                    File("plugins/athena/config.json"),
                )

            sock =
                try {
                    Socket(cfg.slaveAddr, cfg.slavePort)
                } catch (e: Exception) {
                    e.printStackTrace()
                    logger.severe("Could not connect to slave at ${cfg.slaveAddr}:${cfg.slavePort}")
                    server.shutdown()
                    return
                }
            logger.info("Connected to slave at ${cfg.slaveAddr}:${cfg.slavePort}")

            val out = sock!!.getOutputStream()
            Packet.sendPacket(
                out,
                Protocol.PacketServiceConnect.newBuilder().setKey(cfg.key).build(),
            )

            server.scheduler.runTaskAsynchronously(
                this,
                { ->
                    val input = sock!!.getInputStream()
                    try {
                        while (true) {
                            val p = Packet.readPacket(input) ?: break
                            handlePacket(p)
                        }
                    } catch (e: Exception) {
                        if (!shuttingDown) {
                            e.printStackTrace()
                        }
                    }
                    if (!shuttingDown) {
                        logger.info("Connection to slave lost, shutting down")
                        server.shutdown()
                    }
                },
            )
        } catch (e: Exception) {
            logger.severe("Failed to initialize plugin")
            e.printStackTrace()
            server.shutdown()
        }
    }

    override fun onDisable() {
        shuttingDown = true
        sock?.close()
    }

    private fun handlePacket(p: Message) {
        logger.info("Received packet: ${p.javaClass.name}")
    }
}
