package ru.nsu.fit.tcptransferring.server;


import java.io.*;
import java.net.Socket;
import java.nio.charset.StandardCharsets;

public class Loader implements Runnable {
    private final Socket socket;
    private final InputStream in;

    public Loader(Socket socket) throws IOException {
        this.socket = socket;
        this.in = new BufferedInputStream(socket.getInputStream());
    }

    @Override
    public void run() {
        try {
            int nameLength = leBytesToInt();
            byte[] nameBuffer = readFully(nameLength);
            String name = new String(nameBuffer, StandardCharsets.UTF_8);
            System.out.println(name);
        } catch (IOException e) {
            Thread.currentThread().interrupt();
        }

    }

    private int leBytesToInt() throws IOException {
        byte[] b = in.readNBytes(Integer.BYTES);
        if (b.length < 4) throw new IllegalArgumentException("need 4 bytes");
        return (b[0] & 0xFF) |
                ((b[1] & 0xFF) << 8) |
                ((b[2] & 0xFF) << 16) |
                ((b[3] & 0xFF) << 24);
    }

    private byte[] readFully(int len) throws IOException {
        int read = 0;
        byte[] b = new byte[len];
        while (read < len) {
            int count = in.read(b, read, len - read);
            if (count < 0) throw new EOFException();
            read += count;
        }
        return b;
    }
}
