import { useState } from "react";
import DefaultNavigation from "./DefaultNavigation.jsx";
import { useNavigate } from "react-router-dom";

export default function SignUp() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [errorMessage, setErrorMessage] = useState("");

  const navigate = useNavigate();

  const handleSubmit = () => {
    // TODO: data validation on client
    fetch("http://localhost:8080/register", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json; charset=utf-8",
      },
      body: JSON.stringify({
        name: name,
        email: email,
        password: password,
        confirm_password: confirmPassword,
      }),
    }).then((response) => {
      if (response.status == 400) {
        response.json().then((response) => {
          setErrorMessage(response["message"]);
        });
      } else {
        navigate("/");
      }
    });
  };

  return (
    <div name="login-page" className="flex h-[100vh] w-[100vw] flex-col">
      <DefaultNavigation />
      <div className="flex h-full w-full items-center justify-center">
        <div className="flex w-[60%] flex-col items-center justify-around gap-10 bg-[#6B6F80] p-10">
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Name"
            onChange={(e) => setName(e.target.value)}
          />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Email"
            onChange={(e) => setEmail(e.target.value)}
          />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Password"
            type="password"
            onChange={(e) => setPassword(e.target.value)}
          />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Confirm password"
            type="password"
            onChange={(e) => setConfirmPassword(e.target.value)}
          />

          <button
            className="rounded-xl bg-white p-2 text-5xl"
            onClick={() => handleSubmit()}
          >
            Sign Up
          </button>
        </div>
      </div>
    </div>
  );
}
