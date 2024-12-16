package eu.novusmc.athena.velocity

import com.google.inject.Inject
import com.google.protobuf.Message
import com.velocitypowered.api.event.Subscribe
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent
import com.velocitypowered.api.event.proxy.ProxyShutdownEvent
import com.velocitypowered.api.plugin.Dependency
import com.velocitypowered.api.plugin.Plugin
import com.velocitypowered.api.proxy.ProxyServer
import de.pauhull.novus_utils.common.PluginConfig
import eu.novusmc.athena.common.Packet
import eu.novusmc.athena.common.Protocol
import org.slf4j.Logger
import java.io.File
import java.net.Socket
import java.net.SocketException

@Plugin(
    id = "athena",
    version = "0.1.0",
    dependencies = [
        Dependency(id = "kotlin-stdlib")
    ],
)
class AthenaVelocityPlugin @Inject constructor(
    private val server: ProxyServer,
    private val logger: Logger
) {

    private var shuttingDown = false
    private var sock: Socket? = null

    @Subscribe
    fun onProxyInitialization(event: ProxyInitializeEvent) {
        try {
            val cfg = PluginConfig.copyAndLoad(Configuration::class.java, File("plugins/athena/config.json"))

            sock = try {
                Socket(cfg.slaveAddr, cfg.slavePort)
            } catch (e: Exception) {
                e.printStackTrace()
                logger.error("Could not connect to slave at ${cfg.slaveAddr}:${cfg.slavePort}")
                server.shutdown()
                return
            }
            logger.info("Connected to slave at ${cfg.slaveAddr}:${cfg.slavePort}")

            val out = sock!!.getOutputStream()
            Packet.sendPacket(out, Protocol.PacketServiceConnect.newBuilder().setKey(cfg.key).build())

            server.scheduler.buildTask(this, { ->
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
            }).schedule()
        } catch(e: Exception) {
            logger.error("Failed to initialize plugin")
            e.printStackTrace()
            server.shutdown()
        }
    }

    @Subscribe
    fun onProxyShutdown(event: ProxyShutdownEvent) {
        shuttingDown = true
        sock?.close()
    }

    private fun handlePacket(p: Message) {
        logger.info("Received packet: ${p.javaClass.name}")
    }
}