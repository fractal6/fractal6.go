package handlers

import (
	"net/http"
    "github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt" 
)

type Cred struct {
    username string  `json:"name" form:"name" query:"name"`
    password string  `json:"password" form:"password" query:"password"`
}

// HashPassword generates a hash using the bcrypt.GenerateFromPassword                                                                                                                                             
func HashPassword(password string) string {                                                                                                                                                                        
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)                                                                                                                                                 
    if err != nil {                                                                                                                                                                                                
        panic(err)                                                                                                                                                                                                 
    }                                                                                                                                                                                                              
                                                                                                                                                                                                                   
    return string(hash)                                                                                                                                                                                            
} 

// ComparePassword compares the hash                                                                                                                                                                               
func ComparePassword(hash string, password string) bool {                                                                                                                                                          
                                                                                                                                                                                                                   
    if len(password) == 0 || len(hash) == 0 {                                                                                                                                                                      
        return false                                                                                                                                                                                               
    }                                                                                                                                                                                                              
                                                                                                                                                                                                                   
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))                                                                                                                                           
    return err == nil                                                                                                                                                                                              
} 

func Signup(c echo.Context) error {
    c := new(Cred)
    if err := c.Bind(c); err != nil {
        return &echo.HTTPError{
            Code: http.StatusBadRequest, 
            Message: "invalid email or password"}
	}
    
	hashedPassword = HashPassword()


    //return nil
}
