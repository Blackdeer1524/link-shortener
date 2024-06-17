import { useState } from "react";
import { useCookies } from "react-cookie";

import DefaultNavigation from "./DefaultNavigation.jsx";

function App() {
  const [shortURL, setShortURL] = useState("");
  const [longURL, setLongURL] = useState("");

  const [makingRequest, setMakingRequest] = useState(false);

  const [cookies, _, removeCookie] = useCookies(["auth"]);
  console.log(cookies);

  const handleSubmit = () => {
    if (makingRequest) {
      return;
    }

    setMakingRequest(true);
    fetch("http://localhost:8081", {
      method: "POST",
      headers: {
        "Content-Type": "multipart/form-data",
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

  return (
    <div name="page" className="flex h-[100vh] flex-col">
      <DefaultNavigation />
      <div
        name="toolbar"
        className="flex h-full w-full items-center justify-center"
      >
        <div
          name="shortener-box"
          className="flex min-h-[40%] w-[80%] flex-col items-center gap-5 rounded-xl bg-gray-300 p-10"
        >
          <label className="text-xl">URL to shorten</label>
          <input
            type="url"
            id="long-url-input"
            className="w-full rounded p-1 text-xl"
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
