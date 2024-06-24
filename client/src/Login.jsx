import { useState } from "react";
import DefaultNavigation from "./DefaultNavigation.jsx";
import { useNavigate } from "react-router-dom";

export default function SignUp() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const [waitingResponse, setWaitingResponse] = useState(false);
  const [errorMessage, setErrorMessage] = useState("\xa0");

  const navigate = useNavigate();

  const handleSubmit = () => {
    setErrorMessage("\xa0");
    setWaitingResponse(true);

    fetch("http://localhost:8080/login", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json; charset=utf-8",
      },
      body: JSON.stringify({
        email: email,
        password: password,
      }),
    })
      .catch((reason) => {
        setErrorMessage("Couldn't reach server");
        setWaitingResponse(false);
      })
      .then((response) => {
        response.json().then((jResp) => {
          setWaitingResponse(false);
          if (response.status == 200) {
            navigate("/");
          } else {
            setErrorMessage(jResp["message"]);
          }
        });
      });
  };

  return (
    <div name="login-page" className="flex h-[100vh] w-[100vw] flex-col">
      <DefaultNavigation />
      <div className="flex h-full w-full items-center justify-center">
        <div className="flex w-[60%] flex-col items-center justify-around gap-10 bg-[#6B6F80] p-10">
          <p className="text-red-500 text-3xl font-bold whitespace-pre-wrap">
            {errorMessage}
          </p>
          <input
            className="w-full rounded-xl p-2 text-center text-3xl"
            placeholder="Email"
            onChange={(e) => setEmail(e.target.value)}
          />
          <input
            className="w-full rounded-xl p-2 text-center text-3xl"
            placeholder="Password"
            type="password"
            onChange={(e) => setPassword(e.target.value)}
          />

          <button
            className="rounded-xl bg-white p-2 text-3xl"
            onClick={() => handleSubmit()}
            disabled={waitingResponse}
          >
            {waitingResponse ? "Waiting" : "Log in"}
          </button>
        </div>
      </div>
    </div>
  );
}
