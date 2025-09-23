package ru.nsu.fit.tcptransferring.client;


import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.io.*;
import java.net.Socket;

@Component
public class Client {
    private final String pathToFile;
    private final Socket socket;
    private final BufferedReader in;
    private final BufferedOutputStream out;

    public Client(@Value("${file.path}") String pathToFile, @Value("${socket.host}") String host, @Value("${server.port}") int port) throws IOException {
        this.pathToFile = pathToFile;
        socket = new Socket(host, port);
        in = new BufferedReader(new InputStreamReader(socket.getInputStream()));
        out = new BufferedOutputStream(socket.getOutputStream());
        sendFileName();
    }

    private void sendFileName() throws IOException {
        String[] split = pathToFile.split("/");
        String fileName = split[split.length - 1];
        int filenameLength = fileName.length();
        byte[] fileNameBytes = intToLE(filenameLength);
        out.write(fileNameBytes,0,fileNameBytes.length);
        out.flush();
    }

    private byte[] intToLE(int a){
        return  new byte[]{
                (byte)a,
                (byte)(a>>>8),
                (byte)(a>>>16),
                (byte)(a>>>24)
        };


    }
}
