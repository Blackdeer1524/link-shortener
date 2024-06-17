import DefaultNavigation from "./DefaultNavigation.jsx";

export default function Login() {
  return (
    <div name="login-page" className="h-[100vh] w-[100vw] flex flex-col">
      <DefaultNavigation />
      <div className="flex h-full w-full items-center justify-center">
        <div className="flex w-[60%] flex-col items-center justify-around gap-10 bg-[#6B6F80] p-10">
          <input className="w-full rounded-xl p-2 text-center text-5xl" />
          <input
            className="w-full rounded-xl p-2 text-center text-5xl"
            type="password"
          />
          <button className="rounded-xl bg-white p-2 text-5xl">Log in</button>
        </div>
      </div>
    </div>
  );
}
