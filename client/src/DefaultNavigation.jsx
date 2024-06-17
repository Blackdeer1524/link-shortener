import { useNavigate } from "react-router-dom";

function register() {
  fetch("http://localhost:8080/register", {
    credentials: "include",
  })
    .then((response) => {
      return response.json();
    })
    .then((response) => {
      if (response["status"] === 1) {
        alert(response["message"]);
      } else {
        console.log(response);
      }
    });
}

export default function DefaultNavigation() {
  const navigate = useNavigate();

  return (
    <div name="navigation" className="flex h-fit bg-[#6B6F80]">
      <div className="flex flex-grow gap-1">
        <button
          name="log-in"
          className="bg-blue-300 p-2 text-3xl"
          onClick={() => navigate("/")}
        >
          Home
        </button>
      </div>
      <div className="flex flex-grow flex-row-reverse gap-1 bg-[#6B6F80]">
        <button
          name="log-in"
          className="bg-blue-300 p-2 text-3xl"
          onClick={() => navigate("/login")}
        >
          Log in
        </button>
        <button
          name="sign-up"
          className="bg-blue-300 p-2 text-3xl"
          onClick={() => navigate("/signup")}
        >
          Sign up
        </button>
      </div>
    </div>
  );
}
