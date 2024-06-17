import DefaultNavigation from "./DefaultNavigation.jsx";

export default function SignUp() {
  return (
    <div name="login-page" className="flex h-[100vh] w-[100vw] flex-col">
      <DefaultNavigation />
      <div className="flex h-full w-full items-center justify-center">
        <div className="flex w-[60%] flex-col items-center justify-around gap-10 bg-[#6B6F80] p-10">
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Name"
          />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Login"
          />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Email"
          />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            placeholder="Password"
            type="password"
          />

          <button className="rounded-xl bg-white p-2 text-5xl">Sign Up</button>
        </div>
      </div>
    </div>
  );
}
