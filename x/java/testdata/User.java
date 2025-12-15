package com.example.model;

import java.util.UUID; // Import: User IMPORT java.util.UUID

public class User {

    // Contain: User CONTAIN Field (id, username)
    private String id;
    private String username;
    AddonInfo aInfo;

    // Contain: User CONTAIN Field (DEFAULT_ID)
    private static final String DEFAULT_ID = UUID.randomUUID().toString(); // Use: UUID.randomUUID()

    // Contain: User CONTAIN Constructor (User)
    public User(String username) {
        // Use: Field Access (this.id, this.username)
        this.id = DEFAULT_ID;
        this.username = username;
    }

    // Contain: User CONTAIN Method (getId)
    // Return: Method RETURN String Type
    public String getId() {
        return id;
    }

    // Contain: User CONTAIN Method (setUsername)
    // Parameter: Method PARAMETER String Type
    public void setUsername(String username) {
        this.username = username;
    }

    public static class AddonInfo {
        public String name1;
        private String name2;
        protected String name3;
        default String name4;
    }
}