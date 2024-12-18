package eu.novusmc.athena.velocity

import com.google.inject.Inject
import com.google.protobuf.Message
import com.velocitypowered.api.event.Subscribe
import com.velocitypowered.api.event.player.PlayerChooseInitialServerEvent
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent
import com.velocitypowered.api.event.proxy.ProxyShutdownEvent
import com.velocitypowered.api.plugin.Dependency
import com.velocitypowered.api.plugin.Plugin
import com.velocitypowered.api.proxy.ProxyServer
import com.velocitypowered.api.proxy.server.RegisteredServer
import com.velocitypowered.api.proxy.server.ServerInfo
import de.pauhull.novus_utils.common.PluginConfig
import eu.novusmc.athena.common.Configuration
import eu.novusmc.athena.common.Packet
import eu.novusmc.athena.common.Protocol
import java.io.File
import java.net.InetSocketAddress
import java.net.Socket
import org.slf4j.Logger

@Plugin(
    id = "athena",
    version = "0.1.0", // x-release-please-version
    dependencies = [Dependency(id = "kotlin-stdlib")],
)
class AthenaVelocityPlugin
@Inject
constructor(private val server: ProxyServer, private val logger: Logger) {

    private var shuttingDown = false
    private var sock: Socket? = null

    @Subscribe
    fun onProxyInitialization(event: ProxyInitializeEvent) {
        // unregister all servers from velocity.toml
        server.allServers.map(RegisteredServer::getServerInfo).forEach(server::unregisterServer)

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
                    logger.error("Could not connect to slave at ${cfg.slaveAddr}:${cfg.slavePort}")
                    server.shutdown()
                    return
                }
            logger.info("Connected to slave at ${cfg.slaveAddr}:${cfg.slavePort}")

            val out = sock!!.getOutputStream()
            Packet.sendPacket(
                out,
                Protocol.PacketServiceConnect.newBuilder().setKey(cfg.key).build(),
            )

            server.scheduler
                .buildTask(
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
                .schedule()
        } catch (e: Exception) {
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

    @Subscribe
    fun onPlayerChooseInitialServer(event: PlayerChooseInitialServerEvent) {
        event.setInitialServer(server.allServers.firstOrNull())
    }

    private fun handlePacket(p: Message) {
        when (p) {
            is Protocol.PacketProxyRegisterServer -> {
                logger.info("Registering server ${p.serverName} at ${p.host}:${p.port}")
                server.registerServer(
                    ServerInfo(p.serverName, InetSocketAddress.createUnresolved(p.host, p.port))
                )
            }
            is Protocol.PacketProxyUnregisterServer -> {
                logger.info("Unregistering server ${p.serverName}")
                val srv = server.getServer(p.serverName)
                if (srv.isPresent) {
                    server.unregisterServer(srv.get().serverInfo)
                }
            }
        }
    }
}
