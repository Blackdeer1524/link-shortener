import { useState } from "react";
import { useCookies } from "react-cookie";

function App() {
  const [cookies, setCookie, removeCookie] = useCookies(["JWT"]);

  return (
    <div>
      <button
        className="bg-gray-400"
        onClick={() => {
          fetch("http://localhost:8080/register", {
            credentials: "include",
          })
            .then((response) => {
              console.log(response);
              return response.json();
            })
            .then((response) => {
              if (response["status"] === 1) {
                alert(response["message"]);
              } else {
                console.log(response);
              }
            });
        }}
      >
        press me
      </button>
      <button
        className="bg-red-400"
        onClick={() => {
          fetch("http://localhost:8080/cookie", {
            credentials: "include",
          }).then(async (response) => {
            return console.log(await response.text());
          });
        }}
      >
        test cookie
      </button>
    </div>
  );
}

export default App;
