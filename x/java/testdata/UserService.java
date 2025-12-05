package com.example.service;

import com.example.model.User;             // Import: UserService IMPORT com.example.model.User
import com.example.model.NotificationException; // Import: UserService IMPORT com.example.model.NotificationException
import com.example.model.ErrorCode;         // NEW: Import ErrorCode enum
import java.util.List;                     // Import: UserService IMPORT java.util.List

// Annotation: UserService ANNOTATION Target (e.g., Spring/Jakarta Annotation)
@Service("user_service")
public class UserService implements DataService<User> { // Implement: UserService IMPLEMENTS DataService<User>
                                                       // Extend: (Implicitly EXTEND Object)
                                                       // Use: DataService Type, User Type (in generics)

    // Annotation: Field ANNOTATION Target (e.g., Autowired)
    @Autowired
    private UserRepository repository; // Use: UserRepository Type

    // Contain: UserService CONTAIN Method (findById)
    // Parameter: Method PARAMETER String Type
    // Return: Method RETURN User Type
    public User findById(String id) {
        // Call: repository.findOne(id)
        User user = repository.findOne(id);

        // Call: Enum value used in conditional logic
        if (id.equals("404")) {
            // Use: ErrorCode.USER_NOT_FOUND
            System.out.println(ErrorCode.USER_NOT_FOUND.getMessage());
        }

        // Cast: (User) cast expression
        if (user == null) {
            return (User) new Object(); // Create: new Object()
        }
        return user;
    }

    // Contain: UserService CONTAIN Method (createUser)
    // Parameter: Method PARAMETER String Type
    // Return: Method RETURN User Type
    // Throw: Method THROW NotificationException Type
    public User createUser(String name) throws NotificationException {
        // Create: new User(name)
        User newUser = new User(name);

        // Call: repository.save(newUser)
        repository.save(newUser);

        // Throw: Throw new NotificationException(ErrorCode.NAME_EMPTY)
        if (name.isEmpty()) {
            // Use: ErrorCode.NAME_EMPTY
            throw new NotificationException(ErrorCode.NAME_EMPTY); // Create: new NotificationException(...) using Enum
        }

        return newUser;
    }

    // Contain: UserService CONTAIN Method (getAll)
    // Return: Method RETURN List Type (generic)
    @Override
    public List<User> getAll() {
        return repository.findAll(); // Call: repository.findAll()
    }
}

// 假设的泛型接口和仓库接口
interface DataService<T> {
    List<T> getAll(); // Parameter: List Type, Return: List Type
}
interface UserRepository {
    User findOne(String id); // Parameter: String Type, Return: User Type
    void save(User user);    // Parameter: User Type
    List<User> findAll();    // Return: List Type
}
// 假设的注解
@interface Service { String value(); }
@interface Autowired {}