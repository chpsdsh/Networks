package ru.nsu.fit.tcptransferring.server;


import lombok.extern.slf4j.Slf4j;

import java.io.*;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

@Slf4j
public class Loader implements Runnable {
    private final Socket socket;

    public Loader(Socket socket) throws IOException {
        this.socket = socket;
    }

    @Override
    public void run() {
        try (InputStream in = this.socket.getInputStream()) {
            int nameLength = leBytesToInt(in);
            System.out.println(nameLength);
            byte[] nameBuffer = readFully(in, nameLength);
            String name = new String(nameBuffer, StandardCharsets.UTF_8);
            System.out.println(name);
            Path dir = Paths.get("uploads");
            Files.createDirectories(dir);
            Path file = dir.resolve(name);
            if (!Files.exists(file)) {
                Files.createFile(file);
            }
            long fileLength = leBytesToLong(in);
            log.info("after receiving " + fileLength);
        } catch (IOException e) {
            Thread.currentThread().interrupt();
        }

    }

    private int leBytesToInt(InputStream in) throws IOException {
        byte[] b = in.readNBytes(Integer.BYTES);
        int res = 0;
        for (int i = 0; i < Integer.BYTES; i++) {
            res |= (byte) ((b[i] & 0xFF) << 8 * i);
        }
        return res;
    }

    private long leBytesToLong(InputStream in) throws IOException {
        byte[] b = in.readNBytes(Long.BYTES);
        long res = 0;
        for (int i = 0; i < Long.BYTES; i++) {
            res |= (byte) ((b[i] & 0xFF) << 8 * i);
        }
        return res;
    }

    private byte[] readFully(InputStream in, int len) throws IOException {
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
