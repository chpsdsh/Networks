package ru.nsu.fit.tcptransferring.client;


import jakarta.annotation.PostConstruct;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.io.Resource;
import org.springframework.stereotype.Component;

import java.io.*;
import java.net.Socket;
import java.nio.charset.StandardCharsets;

@Slf4j
@Component
public class Client {
    private final Resource fileRes;
    private final String host;
    private final int port;


    public Client(@Value("${file.path}") Resource fileRes, @Value("${socket.host}") String host, @Value("${socket.port}") int port) {
        this.fileRes = fileRes;
        this.host = host;
        this.port = port;
    }

    @PostConstruct
    public void init() throws IOException {
        try (Socket socket = new Socket(host, port);
             InputStream fin = new BufferedInputStream(fileRes.getInputStream());
             BufferedOutputStream out = new BufferedOutputStream(socket.getOutputStream());
             InputStream in = new BufferedInputStream(socket.getInputStream())
        ) {
            sendFileName(out);
            sendFile(out, fin);
            validateResult(in);
        }
    }

    private static void validateResult(InputStream in) throws IOException {
        int res = in.read();
        switch (res) {
            case 0:
                log.info("Sending finished successfully");
                break;
            case 1:
                log.info("Sending finished with failure");
                break;
            default:
                break;
        }
    }

    private void sendFileName(BufferedOutputStream out) throws IOException {
        String fileName = fileRes.getFilename();
        byte[] fileNameBytes = fileName.getBytes(StandardCharsets.UTF_8);
        byte[] leFileNameLength = intToLE(fileNameBytes.length);
        out.write(leFileNameLength, 0, leFileNameLength.length);
        out.write(fileNameBytes, 0, fileNameBytes.length);
        out.flush();
    }

    private void sendFile(BufferedOutputStream out, InputStream fin) throws IOException {
        byte[] fileLengthBytes = longToLE(fileRes.contentLength());
        out.write(fileLengthBytes, 0, fileLengthBytes.length);
        byte[] buf = new byte[64 * 1024];
        int read;
        while ((read = fin.read(buf)) != -1) {
            out.write(buf, 0, read);
        }
        out.flush();
    }

    private byte[] intToLE(int a) {
        byte[] intLE = new byte[Integer.BYTES];
        for (int i = 0; i < Integer.BYTES; i++) {
            intLE[i] = (byte) (a >>> 8 * i);
        }
        return intLE;
    }

    private byte[] longToLE(long a) {
        byte[] longLE = new byte[Long.BYTES];
        for (int i = 0; i < Long.BYTES; i++) {
            longLE[i] = (byte) (a >>> 8 * i);
        }
        return longLE;
    }
}

