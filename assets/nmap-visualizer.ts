type ScanScript = {
  name: string;
  output: string;
};

type ScanPort = {
  port: string;
  protocol: string;
  state: string;
  service: string;
  version: string;
  scripts: ScanScript[];
};

type ScanHost = {
  title: string;
  state: string;
  os: string;
  ports: ScanPort[];
  notes: string[];
  scripts: ScanScript[];
};

function attribute(element: Element | null, name: string): string {
  return element?.getAttribute(name) ?? "";
}

function scriptOutput(element: Element): ScanScript {
  return {
    name: attribute(element, "id") || "NSE-Skript",
    output: attribute(element, "output"),
  };
}

function parseScan(xml: string): ScanHost[] {
  // Nmap's normal external DTD is allowed; custom entities are unnecessary.
  if (/<!ENTITY\s/i.test(xml)) {
    throw new Error("XML mit eigenen Entities wird nicht unterstützt.");
  }
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  if (
    doc.querySelector("parsererror") ||
    doc.documentElement.tagName !== "nmaprun"
  ) {
    throw new Error("Bitte eine gültige Nmap-XML-Datei öffnen.");
  }
  return Array.from(doc.querySelectorAll("nmaprun > host"), (host) => ({
    title: [
      ...new Set(
        Array.from(
          host.querySelectorAll(
            ":scope > address, :scope > hostnames > hostname",
          ),
          (item) =>
            attribute(item, item.tagName === "address" ? "addr" : "name"),
        ).filter(Boolean),
      ),
    ].join(" · ") || "Host ohne Adresse",
    state: attribute(host.querySelector(":scope > status"), "state") ||
      "unbekannt",
    os: attribute(host.querySelector(":scope > os > osmatch"), "name"),
    ports: Array.from(
      host.querySelectorAll(":scope > ports > port"),
      (port) => {
        const service = port.querySelector(":scope > service");
        return {
          port: attribute(port, "portid"),
          protocol: attribute(port, "protocol"),
          state: attribute(port.querySelector(":scope > state"), "state"),
          service: attribute(service, "name"),
          version: ["product", "version", "extrainfo"].map((key) =>
            attribute(service, key)
          ).filter(Boolean).join(" "),
          scripts: Array.from(
            port.querySelectorAll(":scope > script"),
            scriptOutput,
          ),
        };
      },
    ),
    notes: [
      ...Array.from(
        host.querySelectorAll(":scope > ports > extraports"),
        (ports) =>
          `${attribute(ports, "count")} weitere Ports: ${
            attribute(ports, "state")
          }`,
      ),
    ],
    scripts: Array.from(
      host.querySelectorAll(":scope > hostscript > script"),
      scriptOutput,
    ),
  }));
}

// Scan values are untrusted: render them exclusively as text, never as HTML.
function textElement<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  text: string,
): HTMLElementTagNameMap[K] {
  const element = document.createElement(tag);
  element.textContent = text;
  return element;
}

function statusBadge(state: string): HTMLElement {
  let color = "bg-slate-500/15 text-slate-300 ring-slate-500/30";
  let icon = "?";
  switch (state) {
    case "open":
    case "up":
      color = "bg-emerald-500/15 text-emerald-300 ring-emerald-500/30";
      icon = "✓";
      break;
    case "closed":
    case "down":
      color = "bg-rose-500/15 text-rose-300 ring-rose-500/30";
      icon = "×";
      break;
    case "filtered":
    case "open|filtered":
    case "closed|filtered":
      color = "bg-amber-500/15 text-amber-300 ring-amber-500/30";
      icon = "◈";
      break;
    case "unfiltered":
      color = "bg-sky-500/15 text-sky-300 ring-sky-500/30";
      icon = "↔";
  }
  const badge = textElement("span", "");
  badge.className =
    `inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ring-inset ${color}`;
  badge.dataset.state = state;
  const symbol = textElement("span", icon);
  symbol.setAttribute("aria-hidden", "true");
  badge.append(symbol, textElement("span", state || "unbekannt"));
  return badge;
}

function scriptDetails(script: ScanScript): HTMLDetailsElement {
  const details = document.createElement("details");
  details.className =
    "group/nse rounded-lg border border-slate-700/70 bg-slate-950/40";
  const summary = document.createElement("summary");
  summary.className =
    "flex cursor-pointer list-none items-start gap-2 rounded-lg p-3 hover:bg-slate-800/60";
  const arrow = textElement("span", "›");
  arrow.className =
    "text-slate-400 transition-transform group-open/nse:rotate-90";
  arrow.setAttribute("aria-hidden", "true");
  const label = document.createElement("span");
  label.className = "min-w-0 flex-1";
  const name = textElement("span", script.name);
  name.className =
    "block break-words font-mono text-xs font-semibold text-sky-300";
  const preview = textElement(
    "span",
    script.output.trim().split(/\r?\n/)[0] || "Keine Textausgabe.",
  );
  preview.className = "mt-1 block truncate text-xs text-slate-400";
  label.append(name, preview);
  summary.append(arrow, label);
  const toolbar = document.createElement("div");
  toolbar.className =
    "flex items-center gap-3 border-t border-slate-700/60 px-3 py-2";
  const copy = textElement("button", "Kopieren");
  copy.type = "button";
  copy.setAttribute("aria-label", `${script.name} kopieren`);
  copy.className =
    "cursor-pointer rounded-md border border-slate-600 px-2 py-1 text-xs text-slate-300 hover:bg-slate-700 hover:text-white";
  const feedback = textElement("span", "");
  feedback.className = "text-xs text-slate-400";
  feedback.setAttribute("role", "status");
  copy.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(script.output);
      feedback.textContent = "Kopiert.";
    } catch {
      feedback.textContent = "Bitte den Text manuell kopieren.";
    }
  });
  toolbar.append(copy, feedback);
  const output = textElement("pre", script.output || "Keine Textausgabe.");
  output.className =
    "m-0 max-h-64 overflow-auto whitespace-pre p-3 font-mono text-xs leading-relaxed text-slate-200";
  details.append(summary, toolbar, output);
  return details;
}

const viewer = document.querySelector<HTMLElement>(".nmap-viewer");
if (viewer) {
  const fileInput = viewer.querySelector<HTMLInputElement>("#nmap-file")!;
  const search = viewer.querySelector<HTMLInputElement>("#nmap-search")!;
  const state = viewer.querySelector<HTMLSelectElement>("#nmap-state")!;
  const message = viewer.querySelector<HTMLElement>("#nmap-message")!;
  const results = viewer.querySelector<HTMLElement>("#nmap-results")!;
  const summary = viewer.querySelector<HTMLElement>("#nmap-summary")!;
  const matches = viewer.querySelector<HTMLElement>("#nmap-matches")!;
  const hosts = viewer.querySelector<HTMLElement>("#nmap-hosts")!;
  let scan: ScanHost[] = [];
  let loadVersion = 0;

  function render() {
    const query = search.value.trim().toLocaleLowerCase();
    const content = document.createDocumentFragment();
    let count = 0;
    for (const host of scan) {
      const ports = host.ports.filter((port) =>
        !state.value || port.state === state.value
      );
      if (state.value && ports.length === 0) continue;
      const searchable = [
        host.title,
        host.state,
        host.os,
        ...host.notes,
        ...host.scripts.flatMap((script) => [script.name, script.output]),
        ...ports.flatMap((
          port,
        ) => [
          `${port.port}/${port.protocol}`,
          port.state,
          port.service,
          port.version,
          ...port.scripts.flatMap((script) => [script.name, script.output]),
        ]),
      ];
      if (query && !searchable.join(" ").toLocaleLowerCase().includes(query)) {
        continue;
      }
      count++;
      const details = document.createElement("details");
      details.className =
        "group/host overflow-hidden rounded-xl border border-slate-700/70 bg-slate-900/50 shadow-sm";
      const heading = document.createElement("summary");
      heading.className =
        "flex cursor-pointer list-none flex-wrap items-center gap-3 bg-slate-800/50 p-4 hover:bg-slate-800/80";
      const arrow = textElement("span", "›");
      arrow.className =
        "text-slate-400 transition-transform group-open/host:rotate-90";
      arrow.setAttribute("aria-hidden", "true");
      const title = textElement("span", host.title);
      title.className =
        "min-w-0 flex-1 break-words font-mono text-sm font-semibold text-slate-100";
      const portCount = textElement("span", `${ports.length} Ports`);
      portCount.className = "text-xs text-slate-400";
      heading.append(arrow, title, statusBadge(host.state), portCount);
      const hostBody = document.createElement("div");
      hostBody.className = "border-t border-slate-700/60 p-4";
      details.append(heading, hostBody);
      if (host.os) {
        const os = textElement("p", `Betriebssystem: ${host.os}`);
        os.className = "mb-4 text-sm text-slate-400";
        hostBody.append(os);
      }
      if (ports.length) {
        const wrapper = document.createElement("div");
        wrapper.className =
          "overflow-x-auto rounded-lg border border-slate-700/60";
        const table = document.createElement("table");
        table.className = "w-full min-w-[52rem] table-fixed text-left text-sm";
        const caption = table.createCaption();
        caption.className = "sr-only";
        caption.textContent = `Ports für ${host.title}`;
        const header = table.createTHead().insertRow();
        header.className =
          "bg-slate-800/80 text-xs font-semibold uppercase tracking-wide text-slate-400";
        for (
          const [label, width] of [
            ["Port", "w-20"],
            ["Protokoll", "w-24"],
            ["Zustand", "w-44"],
            ["Dienst", "w-32"],
            ["Produkt / Version", "w-36"],
            ["NSE-Ausgaben", ""],
          ]
        ) {
          const cell = textElement("th", label);
          cell.className = `px-4 py-3 ${width}`;
          cell.scope = "col";
          header.append(cell);
        }
        const body = table.createTBody();
        for (const port of ports) {
          const row = body.insertRow();
          row.className =
            "border-t border-slate-700/50 text-slate-300 odd:bg-slate-900/40 hover:bg-slate-800/40";
          for (
            const [value, style] of [
              [port.port, "font-mono font-semibold text-sky-300"],
              [port.protocol, "font-mono text-xs uppercase text-slate-400"],
              ["", ""],
              [port.service, "whitespace-nowrap font-mono text-xs"],
              [
                port.version,
                "break-words text-xs leading-relaxed text-slate-400",
              ],
            ]
          ) {
            const cell = textElement("td", value || "—");
            cell.className = `px-4 py-4 align-top ${style}`;
            row.append(cell);
          }
          row.cells[2].replaceChildren(statusBadge(port.state));
          const scripts = row.insertCell();
          scripts.className = "space-y-2 px-4 py-3 align-top";
          if (port.scripts.length) {
            scripts.append(...port.scripts.map(scriptDetails));
          } else scripts.append(textElement("span", "—"));
        }
        wrapper.append(table);
        hostBody.append(wrapper);
      } else {
        hostBody.append(
          textElement(
            "p",
            "Keine einzeln aufgeführten Ports (z. B. bei einem Discovery-Scan).",
          ),
        );
      }
      for (const note of host.notes) {
        const text = textElement("p", note);
        text.className = "mt-3 text-xs text-slate-400";
        hostBody.append(text);
      }
      if (host.scripts.length) {
        const scripts = document.createElement("div");
        scripts.className = "mt-4 space-y-2";
        const label = textElement("h2", "Host-NSE-Ausgaben");
        label.className = "mb-3 text-sm font-semibold text-slate-300";
        scripts.append(label, ...host.scripts.map(scriptDetails));
        hostBody.append(scripts);
      }
      content.append(details);
    }
    hosts.replaceChildren(content);
    matches.textContent = `${count} von ${scan.length} Hosts angezeigt.`;
  }

  function reset() {
    loadVersion++;
    scan = [];
    fileInput.value = "";
    search.value = "";
    state.value = "";
    hosts.replaceChildren();
    results.hidden = true;
    message.textContent = "Noch keine Datei geöffnet.";
  }

  fileInput.addEventListener("change", async () => {
    const file = fileInput.files?.[0];
    if (!file) return;
    reset();
    const version = loadVersion;
    message.textContent = "Datei wird gelesen …";
    try {
      if (file.size > 10 * 1024 * 1024) {
        throw new Error("Die Datei ist größer als 10 MiB.");
      }
      const xml = await file.text();
      if (version !== loadVersion) return;
      scan = parseScan(xml);
      const ports = scan.flatMap((host) => host.ports);
      const services = new Set(
        ports.map((port) => port.service).filter(Boolean),
      );
      summary.textContent = `${scan.length} Hosts · ${
        ports.filter((port) => port.state === "open").length
      } offene Ports · ${services.size} Dienste`;
      message.textContent = file.name;
      results.hidden = false;
      render();
    } catch (error) {
      if (version !== loadVersion) return;
      message.textContent = error instanceof Error
        ? error.message
        : "Die Datei konnte nicht gelesen werden.";
    }
  });
  search.addEventListener("input", render);
  state.addEventListener("change", render);
  viewer.querySelector("#nmap-clear")!.addEventListener("click", reset);
}
