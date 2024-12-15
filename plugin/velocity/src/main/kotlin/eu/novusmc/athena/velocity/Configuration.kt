package eu.novusmc.athena.velocity

data class Configuration(
    val slaveAddr: String = "127.0.0.1",
    val slavePort: Int = 3000,
    val key: String = ""
)
