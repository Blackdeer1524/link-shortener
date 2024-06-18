import { useState } from "react";
import { useCookies } from "react-cookie";

import DefaultNavigation from "./DefaultNavigation.jsx";
import AuthNavigation from "./AuthNavigation.jsx";

function App() {
  const [shortURL, setShortURL] = useState("");
  const [longURL, setLongURL] = useState("");

  const [makingRequest, setMakingRequest] = useState(false);

  const [cookies, _, removeCookie] = useCookies(["auth"]);

  const handleSubmit = () => {
    if (makingRequest) {
      return;
    }

    setMakingRequest(true);
    fetch("http://localhost:8081", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json; charset=utf-8",
      },
      body: JSON.stringify({
        url: longURL,
      }),
    })
      .catch((reason) => {
        alert(reason);
        setMakingRequest(false);
      })
      .then((response) => response.json())
      .then((response) => {
        setMakingRequest(false);
        setShortURL(response["message"]);
      });
  };

  const navBar = cookies.auth ? (
    <AuthNavigation
      removeAuthCookie={() => {
        removeCookie("auth");
      }}
    />
  ) : (
    <DefaultNavigation />
  );

  return (
    <div name="page" className="flex h-[100vh] flex-col">
      {navBar}
      <div
        name="toolbar"
        className="flex h-full w-full items-center justify-center"
      >
        <div
          name="shortener-box"
          className="flex min-h-[40%] w-[80%] flex-col items-center gap-5 rounded-xl bg-[#6B6F80] p-10"
        >
          <input
            type="url"
            id="long-url-input"
            className="w-full rounded p-1 text-xl"
            placeholder="Url to shorten"
            onChange={(e) => {
              setLongURL(e.target.value);
            }}
          />
          <button
            className="rounded-xl bg-white p-2 text-xl"
            onClick={handleSubmit}
          >
            {makingRequest ? "waiting" : "shorten"}
          </button>
          {shortURL && (
            <>
              <p>{shortURL}</p>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
