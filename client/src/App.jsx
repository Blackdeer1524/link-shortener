import { useState } from "react";
import { useCookies } from "react-cookie";

import DefaultNavigation from "./DefaultNavigation.jsx";
import AuthNavigation from "./AuthNavigation.jsx";

function isValidHttpUrl(string) {
  let url;

  try {
    url = new URL(string);
  } catch (_) {
    return false;
  }

  return url.protocol === "http:" || url.protocol === "https:";
}
function App() {
  const [shortURL, setShortURL] = useState("");
  const [longURL, setLongURL] = useState("");
  const [expiration, setExpiration] = useState(30);
  const [makingRequest, setMakingRequest] = useState(false);

  const [cookies, _, removeCookie] = useCookies(["auth"]);
  const [errorMessage, setErrorMessage] = useState("\xa0");

  const handleSubmit = () => {
    if (makingRequest) {
      return;
    }
    if (!isValidHttpUrl(longURL)) {
      setErrorMessage("Invalid url");
      return
    }

    setMakingRequest(true);
    setErrorMessage("\xa0");
    setShortURL("");

    fetch("http://localhost:8081/create_short_url", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json; charset=utf-8",
      },
      body: JSON.stringify({
        url: longURL,
        expiration: expiration,
      }),
    })
      .catch(() => {
        setErrorMessage("Couldn't reach server");
        setMakingRequest(false);
      })
      .then((response) => {
        setMakingRequest(false);
        if (response.status == 500) {
          response.json().then((rsp) => setErrorMessage(rsp["message"]));
        } else {
          response.json().then((rsp) => setShortURL(rsp["message"]));
        }
      });
  };

  const navBar = cookies.auth ? (
    <AuthNavigation
      removeAuthCookie={() => {
        setErrorMessage("\xa0");
        setShortURL("");
        removeCookie("auth");
      }}
    />
  ) : (
    <DefaultNavigation />
  );

  const chooseExpiration = cookies.auth ? (
    <div className="flex gap-2">
      <input
        type="radio"
        id="choice1"
        name="contact"
        checked={expiration === 30}
        onClick={() => setExpiration(30)}
      />
      <label htmlFor="choice1">30 days</label>

      <input
        type="radio"
        id="choice2"
        name="contact"
        checked={expiration === 90}
        onClick={() => setExpiration(90)}
      />
      <label htmlFor="choice2">90 days</label>

      <input
        type="radio"
        id="choice3"
        name="contact"
        checked={expiration === 365}
        onClick={() => setExpiration(365)}
      />
      <label htmlFor="choice3">365 days</label>
    </div>
  ) : (
    <div></div>
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
          className="flex h-fit w-[80%] flex-col items-center gap-5 rounded-xl bg-[#6B6F80] p-10"
        >
          <p className="block text-xl font-bold text-red-500 whitespace-pre-wrap">
            {errorMessage}
          </p>
          <input
            type="url"
            id="long-url-input"
            className="w-full rounded p-1 text-3xl"
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
          {chooseExpiration}
          {shortURL && (
            <>
              <p className="rounded-xl bg-white p-2 text-xl">{shortURL}</p>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
