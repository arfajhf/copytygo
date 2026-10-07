import { api, ApiError, getSession, type Session, type Dashboard, type UserList, type User } from "./api";
import { layout, escape, welcomeView, authView, dashboardView, accountView, usersView, userFormView } from "./views";
import "./style.css";

const root = document.querySelector<HTMLDivElement>("#app")!;
const path = location.pathname.replace(/\/$/, "") || "/";

function showError(error: unknown): void {
  const box = document.querySelector<HTMLDivElement>("#form-error");
  if (!box) return;
  box.hidden = false;
  box.textContent = error instanceof Error ? error.message : "Request could not be completed.";
  if (error instanceof ApiError) {
    for (const [name, messages] of Object.entries(error.fields)) {
      const field = document.querySelector<HTMLElement>(`[data-error="${CSS.escape(name)}"]`);
      if (field) field.textContent = messages.join(". ");
      document.querySelector<HTMLInputElement>(`[name="${CSS.escape(name)}"]`)?.setAttribute("aria-invalid", "true");
    }
  }
}
function bindForm(id: string, submit: (data: Record<string, string>) => Promise<void>): void {
  const form = document.querySelector<HTMLFormElement>(`#${id}`);
  form?.addEventListener("submit", async event => {
    event.preventDefault();
    const button = form.querySelector<HTMLButtonElement>('button[type="submit"]')!;
    button.disabled = true;
    form.querySelectorAll(".field-error").forEach(node => { node.textContent = ""; });
    form.querySelectorAll("[aria-invalid]").forEach(node => node.removeAttribute("aria-invalid"));
    const notice = document.querySelector<HTMLDivElement>("#form-error");
    if (notice) notice.hidden = true;
    try {
      await submit(Object.fromEntries(Array.from(new FormData(form).entries()).map(([name, value]) => [name, String(value)])));
    } catch (error) { showError(error); } finally { button.disabled = false; }
  });
}
async function start(): Promise<void> {
  const session = await getSession();
  document.title = session.appName;
  const protectedPage = !["/", "/login", "/register"].includes(path);
  if (protectedPage && !session.user) { location.replace("/login"); return; }
  if (session.user && ["/login", "/register"].includes(path)) { location.replace("/dashboard"); return; }
  const admin = session.multi && session.user?.role === "admin";
  if (path.startsWith("/users") && !admin) {
    root.innerHTML = layout(session, "Access denied", '<section class="card"><h1>Access denied</h1><p>Administrator access is required.</p><a href="/dashboard">Return to dashboard</a></section>');
  } else if (path === "/") {
    root.innerHTML = welcomeView(session);
  } else if (path === "/login" || path === "/register") {
    const register = path === "/register";
    root.innerHTML = authView(session, register);
    bindForm("auth-form", async data => {
      await api(`/api/session/${register ? "register" : "login"}`, "POST", data);
      location.assign("/dashboard");
    });
  } else if (path === "/dashboard") {
    root.innerHTML = dashboardView(session, await api<Dashboard>("/api/dashboard"));
  } else if (path === "/account") {
    root.innerHTML = accountView(session);
  } else if (path === "/users") {
    root.innerHTML = usersView(session, await api<UserList>(`/api/users${location.search}`));
    document.querySelectorAll<HTMLButtonElement>("[data-delete]").forEach(button => {
      button.addEventListener("click", async () => {
        if (!confirm(`Delete ${button.dataset.name}? This cannot be undone.`)) return;
        button.disabled = true;
        try { await api(`/api/users/${encodeURIComponent(button.dataset.delete!)}`, "DELETE"); location.assign("/users?saved=deleted"); }
        catch (error) { showError(error); button.disabled = false; }
      });
    });
  } else if (path === "/users/create" || /^\/users\/\d+\/edit$/.test(path)) {
    const id = path === "/users/create" ? undefined : path.split("/")[2];
    const user = id ? await api<User>(`/api/users/${id}`) : undefined;
    root.innerHTML = userFormView(session, user);
    bindForm("user-form", async data => {
      await api(id ? `/api/users/${id}` : "/api/users", id ? "PUT" : "POST", data);
      location.assign(`/users?saved=${id ? "updated" : "created"}`);
    });
  } else {
    root.innerHTML = layout(session, "Not found", '<section class="card"><h1>Page not found</h1><a href="/">Return home</a></section>');
  }
  bindLogout(session);
}
function bindLogout(_session: Session): void {
  document.querySelector<HTMLButtonElement>("#logout")?.addEventListener("click", async event => {
    const button = event.currentTarget as HTMLButtonElement;
    button.disabled = true;
    try { await api("/api/session/logout", "POST"); location.assign("/login"); }
    catch (error) { button.disabled = false; alert(error instanceof Error ? error.message : "Could not log out."); }
  });
}
start().catch(error => {
  if (error instanceof ApiError && error.status === 401 && path !== "/login") { location.replace("/login"); return; }
  root.innerHTML = `<main class="main"><section class="card"><h1>Couldn't load this page</h1><p>${escape(error instanceof Error ? error.message : "Please try again.")}</p><button class="button" id="retry">Try again</button></section></main>`;
  document.querySelector("#retry")?.addEventListener("click", () => location.reload());
});
