package eu.novusmc.athena.velocity

import com.google.inject.Inject
import com.google.protobuf.Message
import com.velocitypowered.api.event.Subscribe
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent
import com.velocitypowered.api.plugin.Dependency
import com.velocitypowered.api.plugin.Plugin
import com.velocitypowered.api.proxy.ProxyServer
import de.pauhull.novus_utils.common.PluginConfig
import eu.novusmc.athena.common.Packet
import eu.novusmc.athena.common.Protocol
import org.slf4j.Logger
import java.io.File
import java.net.Socket

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

    @Subscribe
    fun onProxyInitialization(event: ProxyInitializeEvent) {
        val cfg = PluginConfig.copyAndLoad(Configuration::class.java, File("plugins/athena/config.json"))

        val sock = try {
            Socket(cfg.slaveAddr, cfg.slavePort)
        } catch (e: Exception) {
            logger.error("Could not connect to slave at ${cfg.slaveAddr}:${cfg.slavePort}")
            return
        }

        val out = sock.getOutputStream()
        Packet.sendPacket(out, Protocol.PacketServiceConnect.newBuilder().build())

        server.scheduler.buildTask(this, { ->
            val input = sock.getInputStream()
            while (sock.isConnected) {
                try {
                    val p = Packet.readPacket(input)
                    handlePacket(p)
                } finally {
                    break
                }
            }
            server.shutdown()
        }).schedule()
    }

    private fun handlePacket(p: Message) {
        logger.info("Received packet: ${p.javaClass.name}")
    }
}