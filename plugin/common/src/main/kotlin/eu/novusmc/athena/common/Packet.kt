package eu.novusmc.athena.common

import com.google.protobuf.Any
import com.google.protobuf.Message
import java.io.EOFException
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

    fun readPacket(input: InputStream): Message? {
        val len = readBigEndian32(input)
        val buf = readExactly(input, len)
        val env = Protocol.Envelope.parseFrom(buf)
        val typeUrl = env.payload.typeUrl
        val typeName = typeUrl.substringAfter("type.googleapis.com/")
        val className = typeName.split(".").joinToString("$").replaceFirstChar(Char::uppercase)
        val clazz = Class.forName("eu.novusmc.athena.common.$className") as Class<Message>
        val msg = env.payload.unpack(clazz)
        return msg
    }

    private fun readExactly(input: InputStream, len: Int): ByteArray {
        val buf = ByteArray(len)
        var read = 0
        while (read < len) {
            val n = input.read(buf, read, len - read)
            if (n == -1) {
                throw EOFException()
            }
            read += n
        }
        return buf
    }

    private fun writeBigEndian32(out: OutputStream, i: Int) {
        out.write(ByteBuffer.allocate(4).order(ByteOrder.BIG_ENDIAN).putInt(i).array())
    }

    private fun readBigEndian32(input: InputStream): Int {
        val buf = readExactly(input, 4)
        return ByteBuffer.wrap(buf).order(ByteOrder.BIG_ENDIAN).int
    }
}
