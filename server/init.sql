CREATE TABLE Users (
    Id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    Name VarChar(300) NOT NULL,
    Email VarChar(80)  NOT NULL,
    HashedPassword CHAR(60) NOT NULL
)
;

CREATE UNIQUE INDEX users_unique_emails ON Users(Email) INCLUDE(HashedPassword)
;

--- this uuid is reserved for anonymous users
INSERT INTO Users(Id, Name, Email, HashedPassword) 
    VALUES ('db092ed4-306a-4d4f-be5f-fd2f1487edbe', 'dummy value', 'dumy value', 'dummy value')
;

CREATE TABLE Urls (
	ShortUrl VarChar(5) PRIMARY KEY,
	LongUrl VarChar(300) NOT NULL,
	UserId uuid NOT NULL references Users(Id),
	ExpirationDate Timestamp NOT NULL DEFAULT now() + interval '30' day
)
;
