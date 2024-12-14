package eu.novusmc.athena.common

import com.google.protobuf.Any
import com.google.protobuf.Message
import java.io.InputStream
import java.io.OutputStream
import java.nio.ByteBuffer
import java.nio.ByteOrder

object Packet {

    fun sendPacket(out: OutputStream, packet: Message) {
        val payload = Any.pack(packet)
        Protocol.Envelope.newBuilder().setPayload(payload).build()
        val env = Protocol.Envelope.newBuilder().setPayload(payload).build()
        val buf = env.toByteArray()
        writeBigEndian32(out, buf.size)
        out.write(buf)
    }

    fun readPacket(input: InputStream): Message {
        val len = readBigEndian32(input)
        val buf = input.readNBytes(len)
        val env = Protocol.Envelope.parseFrom(buf)
        val typeUrl = env.payload.typeUrl
        val typeName = typeUrl.substringAfter("type.googleapis.com/")
        val clazz = Class.forName("eu.novusmc.athena.common.Protocol\$${typeName}") as Class<Message>
        val msg = env.payload.unpack(clazz)
        return msg
    }

    private fun writeBigEndian32(out: OutputStream, i: Int) {
        out.write(ByteBuffer.allocate(4).order(ByteOrder.BIG_ENDIAN).putInt(i).array())
    }

    private fun readBigEndian32(input: InputStream): Int {
        val buf = ByteArray(4)
        input.read(buf)
        return ByteBuffer.wrap(buf).order(ByteOrder.BIG_ENDIAN).int
    }

}