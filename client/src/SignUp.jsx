import { useState } from "react";
import DefaultNavigation from "./DefaultNavigation.jsx";
import { useNavigate } from "react-router-dom";

const validateEmail = (email) => {
  return String(email)
    .toLowerCase()
    .match(
      /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|.(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/,
    );
};

export default function SignUp() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [waitingResponse, setWaitingResponse] = useState(false);
  const [errorMessage, setErrorMessage] = useState("\xa0");

  const navigate = useNavigate();

  const handleSubmit = () => {
    let error = "";
    if (name.length == 0) {
      error += "* Name field is required\n";
    }
    if (name.length > 300) {
      error +=
        "* Name field is too long (at most 300 characters are permitted)\n";
    }
    if (!validateEmail(email) || email.length > 60) {
      error += "* Invalid email\n";
    }
    if (password.length < 8) {
      error += "* Password is too short (at least 8 characters are required)\n";
    }
    if (password.length > 64) {
      error += "* Password is too long (at most 64 characters are permitted)\n";
    }

    if (password !== confirmPassword) {
      error += "* Passwords have to match\n";
    }
    if (error) {
      setErrorMessage(error);
      return;
    }

    setErrorMessage("\xa0");
    setWaitingResponse(true);

    fetch("http://localhost:8080/signup", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json; charset=utf-8",
      },
      body: JSON.stringify({
        name: name,
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
          <p className="text-red-500 text-xl font-bold whitespace-pre-wrap">
            {errorMessage}
          </p>
          <input
            className="w-full rounded-xl p-2 text-center text-3xl"
            placeholder="Name"
            onChange={(e) => setName(e.target.value)}
          />
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
          <input
            className="w-full rounded-xl p-2 text-center text-3xl"
            placeholder="Confirm password"
            type="password"
            onChange={(e) => setConfirmPassword(e.target.value)}
          />

          <button
            className="rounded-xl bg-white p-2 text-5xl"
            onClick={() => handleSubmit()}
            disabled={waitingResponse}
          >
            {waitingResponse ? "Waiting" : "Sign up"}
          </button>
        </div>
      </div>
    </div>
  );
}
