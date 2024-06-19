import { useEffect, useState } from "react";
import { useCookies } from "react-cookie";
import { useNavigate } from "react-router-dom";
import AuthNavigation from "./AuthNavigation";

export default function Profile() {
  const [cookies, _, removeCookie] = useCookies(["auth"]);
  const [history, setHistory] = useState([]);

  const navigate = useNavigate();

  useEffect(() => {
    if (!cookies.auth) {
      return;
    }
    fetch("http://localhost:8082/history", {
      method: "GET",
      credentials: "include",
    })
      .catch()
      .then((response) => {
        response.json().then((jRsp) => {
          setHistory(jRsp);
        });
      });
  }, [cookies.auth]);

  if (!cookies.auth) {
    return navigate("/");
  }

  return (
    <div name="profile" className="flex h-[100vh] w-[100vw] flex-col">
      <AuthNavigation
        removeAuthCookie={() => {
          removeCookie("auth");
          navigate("/");
        }}
      />
      <div className="flex h-full w-full items-center justify-center overflow-auto">
        <div className="flex w-fit flex-col items-center justify-around bg-[#6B6F80] p-10">
          <table>
            <tr>
              <th>Short URL</th>
              <th>Long URL</th>
              <th>Expiration Date</th>
            </tr>
            {history.map((row) => {
              return (
                <tr key={row["short_url"]}>
                  <td>{row["short_url"]}</td>
                  <td>{row["long_url"]}</td>
                  <td>{row["expiration_date"]}</td>
                </tr>
              );
            })}
          </table>
        </div>
      </div>
    </div>
  );
}
