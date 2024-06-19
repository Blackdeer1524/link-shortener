CREATE TABLE Users (
    Id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    Name VarChar(300) NOT NULL,
    Email VarChar(80)  NOT NULL,
    HashedPassword CHAR(60) NOT NULL
)
;

CREATE UNIQUE INDEX users_unique_emails ON Users(Email) INCLUDE(HashedPassword)
;

CREATE TABLE Urls (
	ShortUrl VarChar(5) PRIMARY KEY,
	LongUrl VarChar(300) NOT NULL,
	UserId uuid NOT NULL references Users(Id),
	ExpirationDate Timestamp NOT NULL DEFAULT now() + interval '30' day
)
;
