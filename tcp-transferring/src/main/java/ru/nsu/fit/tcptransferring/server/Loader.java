package ru.nsu.fit.tcptransferring.server;


import lombok.extern.slf4j.Slf4j;

import java.io.*;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;

@Slf4j
public class Loader implements Runnable {
    private final Socket socket;
    private final ExecutorService executor;
    private final AtomicLong totalReceived = new AtomicLong(0);
    private final AtomicLong currentReceived = new AtomicLong(0);
    private static final int INTERVAL_MILLIS = 3000;
    private static final double MILLIS_TO_SECONDS = 1000.0;
    private long startTime;

    public Loader(Socket socket) {
        this.socket = socket;
        this.executor = Executors.newSingleThreadExecutor();
    }

    @Override
    public void run() {
        try (InputStream in = this.socket.getInputStream();
             OutputStream out = this.socket.getOutputStream()) {
            int nameLength = leBytesToInt(in);
            byte[] nameBuffer = readFully(in, nameLength);
            String name = new String(nameBuffer, StandardCharsets.UTF_8);
            Path dir = Paths.get("uploads");
            Files.createDirectories(dir);
            Path file = dir.resolve(name);
            if (!Files.exists(file)) {
                Files.createFile(file);
            }
            long fileLength = leBytesToLong(in);
            readFileData(in, fileLength, file);
            calculateAndSendResult(out, fileLength, totalReceived.get());
            log.info("Finished loading");
            closeAll();
        } catch (IOException e) {
            Thread.currentThread().interrupt();
        }
    }

    private int leBytesToInt(InputStream in) throws IOException {
        byte[] b = in.readNBytes(Integer.BYTES);
        int res = 0;
        for (int i = 0; i < Integer.BYTES; i++) {
            res |= (b[i] & 0xFF) << (8 * i);
        }
        return res;
    }

    private long leBytesToLong(InputStream in) throws IOException {
        byte[] b = in.readNBytes(Long.BYTES);
        if (b.length != Long.BYTES) throw new EOFException("Not enough bytes for long");
        long res = 0L;
        for (int i = 0; i < Long.BYTES; i++) {
            res |= ((long) (b[i] & 0xFF)) << (8 * i);
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

    private void readFileData(InputStream in, long dataLength, Path path) throws IOException {
        try (OutputStream out = new BufferedOutputStream(Files.newOutputStream(
                path, StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING))) {
            byte[] buf = new byte[1024 * 64];
            long remaining = dataLength;
            showMetrics();
            while (remaining > 0) {
                long toRead = Math.min(remaining, buf.length);
                int read = in.read(buf, 0, (int) toRead);
                if (read == -1) {
                    throw new EOFException();
                }
                out.write(buf, 0, read);
                totalReceived.addAndGet(read);
                currentReceived.addAndGet(read);
                remaining -= read;
            }
        } catch (IOException e) {
            log.error("Error while reading file", e);
            closeAll();
        }
    }

    private void calculateAndSendResult(OutputStream out, long expected, long received) throws IOException {
        if (expected == received) {
            out.write(0);
        } else {
            out.write(1);
        }
        out.flush();
    }

    private void showMetrics() {
        startTime = System.currentTimeMillis();
        executor.execute(() -> {
            while (!Thread.currentThread().isInterrupted()) {
                try {
                    currentReceived.set(0);
                    Thread.sleep(INTERVAL_MILLIS);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                }
                log.info("Total speed: " + (double) totalReceived.get() / (System.currentTimeMillis() - startTime) * MILLIS_TO_SECONDS);
                log.info("Current speed: " + (double) currentReceived.get() / INTERVAL_MILLIS * MILLIS_TO_SECONDS);
            }
        });
    }

    private void closeAll() throws IOException {
        socket.close();
        executor.shutdownNow();
        if (totalReceived.get() != 0) {
            log.info("Final total speed:" + (double) totalReceived.get() / (System.currentTimeMillis() - startTime) * MILLIS_TO_SECONDS);
        }
    }
}


