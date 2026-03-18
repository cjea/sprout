const panelRoot = document.querySelector("[data-panel-root]");
const panelViews = Array.from(document.querySelectorAll("[data-panel-view]"));
const openButtons = Array.from(document.querySelectorAll("[data-panel]"));
const closeButtons = Array.from(document.querySelectorAll("[data-close-panel]"));
const caseName = document.querySelector("[data-case-name]");
const identityMeta = document.querySelector("[data-identity-meta]");
const previousPassageButton = document.querySelector("[data-previous-passage]");
const progressChip = document.querySelector("[data-progress-chip]");
const passageHeading = document.querySelector("[data-passage-heading]");
const passageMeta = document.querySelector("[data-passage-meta]");
const passageText = document.querySelector("[data-passage-text]");
const citationAnchor = document.querySelector("[data-citation-anchor]");
const citationList = document.querySelector("[data-citation-list]");
const citationButton = document.querySelector("[data-citation-button]");
const continueButton = document.querySelector("[data-continue-button]");
const questionAnchor = document.querySelector("[data-question-anchor]");
const questionInput = document.querySelector("#question-input");
const questionSubmit = document.querySelector("[data-question-submit]");
const answerAnchor = document.querySelector("[data-answer-anchor]");
const answerBody = document.querySelector("[data-answer-body]");
const answerEvidence = document.querySelector("[data-answer-evidence]");
const answerCaveat = document.querySelector("[data-answer-caveat]");
const repairButton = document.querySelector("[data-repair-button]");
const repairAnchor = document.querySelector("[data-repair-anchor]");
const repairIssues = document.querySelector("[data-repair-issues]");
const repairHistory = document.querySelector("[data-repair-history]");
const repairUndo = document.querySelector("[data-repair-undo]");
const repairActionButtons = Array.from(document.querySelectorAll("[data-repair-action]"));
const footerCopy = document.querySelector("[data-footer-copy]");
let readerState = null;
let selectionState = { start: 0, end: 0, quote: "" };
let answerState = null;

function currentPassageIndex() {
  if (!readerState) {
    return -1;
  }
  return readerState.passages.findIndex((passage) => passage.passageId === readerState.passage.passageId);
}

function previousPassageId() {
  const index = currentPassageIndex();
  if (index <= 0) {
    return "";
  }
  return readerState.passages[index - 1].passageId;
}

async function openPassage(passageId, { push = true, closePanel = false } = {}) {
  const params = currentParams();
  params.set("passage", passageId);
  if (closePanel) {
    params.delete("panel");
    params.delete("citation");
    params.delete("question");
  }
  params.delete("start");
  params.delete("end");
  selectionState = { start: 0, end: 0, quote: "" };
  answerState = null;
  if (push) {
    pushParams(params);
  } else {
    replaceParams(params);
  }
  await loadReader();
}

function currentParams() {
  return new URLSearchParams(window.location.search);
}

function replaceParams(params) {
  const query = params.toString();
  const url = query ? `${window.location.pathname}?${query}` : window.location.pathname;
  window.history.replaceState({}, "", url);
}

function pushParams(params) {
  const query = params.toString();
  const url = query ? `${window.location.pathname}?${query}` : window.location.pathname;
  window.history.pushState({}, "", url);
}

function setPanel(name, { push = false, citationId = "", questionId = "" } = {}) {
  const open = Boolean(name);
  panelRoot.hidden = !open;

  for (const view of panelViews) {
    view.hidden = view.dataset.panelView !== name;
  }

  const params = currentParams();
  if (open) {
    params.set("panel", name);
  } else {
    params.delete("panel");
    params.delete("citation");
    params.delete("question");
  }
  if (citationId) {
    params.set("citation", citationId);
  } else {
    params.delete("citation");
  }
  if (questionId) {
    params.set("question", questionId);
  } else if (name !== "answer") {
    params.delete("question");
  }
  if (selectionState.quote) {
    params.set("start", String(selectionState.start));
    params.set("end", String(selectionState.end));
  } else {
    params.delete("start");
    params.delete("end");
  }
  if (push) {
    pushParams(params);
  } else {
    replaceParams(params);
  }
}

for (const button of openButtons) {
  button.addEventListener("click", () => {
    setPanel(button.dataset.panel, { push: true });
  });
}

for (const button of closeButtons) {
  button.addEventListener("click", () => {
    setPanel("", { push: true });
  });
}

async function loadReader() {
  const params = currentParams();

  try {
    const query = params.toString();
    const response = await fetch(query ? `/api/reader?${query}` : "/api/reader");
    if (!response.ok) {
      throw new Error(`reader request failed: ${response.status}`);
    }

    const data = await response.json();
    readerState = { ...data };
    caseName.textContent = data.opinion.caseName;
    identityMeta.textContent = `${data.opinion.docket} · ${data.passage.sectionId} · ${data.passages.length} passages prepared`;
    progressChip.textContent = data.passage.passageId;
    passageHeading.textContent = `Section ${data.passage.sectionId}`;
    passageMeta.textContent = `Pages ${data.passage.pageStart}-${data.passage.pageEnd} · ${data.passage.citations.length} citations in view`;
    passageText.textContent = data.passage.text;
    renderCitationPanel(params.get("citation"));
    restoreSelection(params);
    renderQuestionPanel();
    renderAnswerPanel();
    renderRepairPanel();
    citationList.innerHTML = "";
    for (const citation of data.passage.citations) {
      const item = document.createElement("li");
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = `${citation.kind}: ${citation.rawText}`;
      button.addEventListener("click", () => {
        renderCitationPanel(citation.citationId);
        setPanel("citation", { push: true, citationId: citation.citationId });
      });
      item.appendChild(button);
      citationList.appendChild(item);
    }
    citationButton.disabled = data.passage.citations.length === 0;
    const passageIndex = currentPassageIndex();
    const previousId = previousPassageId();
    if (previousId) {
      previousPassageButton.hidden = false;
      previousPassageButton.textContent = previousId;
      previousPassageButton.disabled = false;
    } else {
      previousPassageButton.hidden = true;
      previousPassageButton.textContent = "Previous";
      previousPassageButton.disabled = true;
    }
    const hasNextPassage = passageIndex >= 0 && passageIndex + 1 < data.passages.length;
    continueButton.disabled = !hasNextPassage;
    continueButton.textContent = hasNextPassage ? "Continue" : "End Of Opinion";
    footerCopy.textContent = `Current passage ${data.passage.passageId}. ${data.progress.completedPassages.length} completed.`;
    normalizePassageParams();
    restorePanelFromURL();
  } catch (error) {
    readerState = null;
    caseName.textContent = "Browser reader failed to load";
    identityMeta.textContent = "The shell is up, but the reader API did not return data.";
    progressChip.textContent = "Load error";
    previousPassageButton.hidden = true;
    previousPassageButton.disabled = true;
    passageHeading.textContent = "Reader unavailable";
    passageMeta.textContent = "Passage metadata unavailable.";
    passageText.textContent = String(error);
    citationAnchor.textContent = "No citation detail available.";
    citationList.innerHTML = "";
    citationButton.disabled = true;
    if (repairButton) {
      repairButton.disabled = true;
    }
    repairUndo.disabled = true;
    continueButton.disabled = true;
    continueButton.textContent = "Continue";
    questionSubmit.disabled = true;
    footerCopy.textContent = "Keep the passage primary even in failure states.";
  }
}

async function completeCurrentPassage() {
  if (!readerState) {
    return;
  }
  await fetch("/api/complete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      userId: readerState.progress.userId,
      opinionId: readerState.opinion.opinionId,
      passageId: readerState.passage.passageId,
    }),
  });
}

continueButton.addEventListener("click", async () => {
  if (!readerState) {
    return;
  }
  const passageIndex = currentPassageIndex();
  if (passageIndex < 0 || passageIndex + 1 >= readerState.passages.length) {
    return;
  }
  await completeCurrentPassage();
  const nextPassageId = readerState.passages[passageIndex + 1].passageId;
  await openPassage(nextPassageId, { push: true, closePanel: true });
});

previousPassageButton.addEventListener("click", async () => {
  const previousId = previousPassageId();
  if (!previousId) {
    return;
  }
  await openPassage(previousId, { push: true });
});

function renderCitationPanel(citationId) {
  if (!readerState) {
    citationAnchor.textContent = "No active citation selected.";
    return;
  }
  const citations = readerState.passage.citations;
  const active = citations.find((citation) => citation.citationId === citationId) || citations[0];
  citationAnchor.textContent = active ? active.rawText : "No citation in this passage.";
}

function renderQuestionPanel() {
  if (selectionState.quote) {
    questionAnchor.textContent = `“${selectionState.quote}”`;
  } else if (readerState) {
    const quote = readerState.passage.text.slice(0, Math.min(96, readerState.passage.text.length));
    questionAnchor.textContent = `“${quote}”`;
  } else {
    questionAnchor.textContent = "Selected quote stays visible here.";
  }
  questionSubmit.disabled = !readerState;
}

function renderAnswerPanel() {
  if (!answerState) {
    answerAnchor.textContent = selectionState.quote ? `“${selectionState.quote}”` : "Selected quote preserved here.";
    answerBody.textContent = "Answer review stays inside the workspace instead of navigating away from reading.";
    answerEvidence.innerHTML = "";
    answerCaveat.textContent = "Heuristic placeholder. Real answer flow is loading from the browser API when used.";
    return;
  }

  answerAnchor.textContent = `“${answerState.question.quote}”`;
  answerBody.textContent = answerState.answer.answer;
  answerEvidence.innerHTML = "";
  for (const evidence of answerState.answer.evidence) {
    const item = document.createElement("li");
    item.textContent = `${evidence.label}: ${evidence.quote}`;
    answerEvidence.appendChild(item);
  }
  answerCaveat.textContent = answerState.answer.caveats.join(" ") || `Model: ${answerState.answer.modelName}`;
}

function renderRepairPanel() {
  if (!readerState) {
    repairAnchor.textContent = "Repair tools unavailable.";
    repairIssues.innerHTML = "";
    repairHistory.innerHTML = "";
    repairUndo.disabled = true;
    for (const button of repairActionButtons) {
      button.disabled = true;
    }
    return;
  }

  repairAnchor.textContent = `Passage ${readerState.passage.passageId}. Direct structural changes only.`;
  repairIssues.innerHTML = "";
  for (const issue of readerState.repair.issues) {
    const item = document.createElement("li");
    item.textContent = `${issue.kind}: ${issue.summary}`;
    repairIssues.appendChild(item);
  }
  if (!readerState.repair.issues.length) {
    const item = document.createElement("li");
    item.textContent = "No classified issue for this passage right now.";
    repairIssues.appendChild(item);
  }

  repairHistory.innerHTML = "";
  for (const entry of readerState.repair.history.slice().reverse()) {
    const item = document.createElement("li");
    item.textContent = `r${entry.revision} ${entry.operationKind} on ${entry.targetPassage}`;
    repairHistory.appendChild(item);
  }
  if (!readerState.repair.history.length) {
    const item = document.createElement("li");
    item.textContent = "No persisted repair history yet.";
    repairHistory.appendChild(item);
  }

  const actionState = {
    mergeNext: readerState.repair.canMergeNext,
    mergePrevious: readerState.repair.canMergePrevious,
    splitSentence: readerState.repair.canSplitSentence,
    removeHeader: readerState.repair.canRemoveHeader,
  };
  for (const button of repairActionButtons) {
    button.disabled = !actionState[button.dataset.repairAction];
  }
  repairUndo.disabled = readerState.repair.history.length === 0;
  if (repairButton) {
    repairButton.disabled = false;
  }
}

function restoreSelection(params) {
  const start = Number(params.get("start"));
  const end = Number(params.get("end"));
  if (!Number.isNaN(start) && !Number.isNaN(end) && readerState && start >= 0 && end > start && end <= readerState.passage.text.length) {
    selectionState = {
      start,
      end,
      quote: readerState.passage.text.slice(start, end),
    };
  } else {
    selectionState = { start: 0, end: 0, quote: "" };
  }
}

function normalizePassageParams() {
  if (!readerState) {
    return;
  }
  const params = currentParams();
  params.set("passage", readerState.passage.passageId);
  replaceParams(params);
}

function restorePanelFromURL() {
  const params = currentParams();
  const panel = params.get("panel");
  const validPanels = new Set(["citation", "question", "answer", "repair"]);
  if (!panel || !validPanels.has(panel)) {
    setPanel("", { push: false });
    return;
  }
  setPanel(panel, {
    push: false,
    citationId: params.get("citation") || "",
    questionId: params.get("question") || "",
  });
}

passageText.addEventListener("mouseup", () => {
  if (!readerState) {
    return;
  }
  const selection = window.getSelection();
  if (!selection || selection.isCollapsed || !passageText.contains(selection.anchorNode) || !passageText.contains(selection.focusNode)) {
    return;
  }
  const range = selection.getRangeAt(0);
  const selectedText = range.toString().trim();
  if (!selectedText) {
    return;
  }
  const fullText = passageText.textContent;
  const start = fullText.indexOf(selectedText);
  if (start < 0) {
    return;
  }
  selectionState = {
    start,
    end: start + selectedText.length,
    quote: selectedText,
  };
  renderQuestionPanel();
  renderAnswerPanel();
  const params = currentParams();
  params.set("start", String(selectionState.start));
  params.set("end", String(selectionState.end));
  replaceParams(params);
});

questionSubmit.addEventListener("click", async () => {
  if (!readerState) {
    return;
  }
  const selected = selectionState.quote
    ? selectionState
    : {
        start: 0,
        end: Math.min(readerState.passage.text.length, 96),
        quote: readerState.passage.text.slice(0, Math.min(readerState.passage.text.length, 96)),
      };
  const questionTextValue = questionInput.value.trim();
  if (!questionTextValue) {
    questionInput.focus();
    return;
  }

  const response = await fetch("/api/question", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      userId: readerState.progress.userId,
      opinionId: readerState.opinion.opinionId,
      passageId: readerState.passage.passageId,
      start: selected.start,
      end: selected.end,
      text: questionTextValue,
    }),
  });
  if (!response.ok) {
    answerState = null;
    answerBody.textContent = `Question failed: ${response.status}`;
    setPanel("answer", { push: true });
    return;
  }
  answerState = await response.json();
  renderAnswerPanel();
  const params = currentParams();
  params.set("question", answerState.question.questionId);
  pushParams(params);
  setPanel("answer", { push: false, questionId: answerState.question.questionId });
  await loadReader();
});

for (const button of repairActionButtons) {
  button.addEventListener("click", async () => {
    if (!readerState) {
      return;
    }
    const response = await fetch("/api/repair/apply", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        userId: readerState.progress.userId,
        opinionId: readerState.opinion.opinionId,
        passageId: readerState.passage.passageId,
        operation: button.dataset.repairAction,
      }),
    });
    if (!response.ok) {
      const message = await response.text();
      repairAnchor.textContent = `Repair failed: ${response.status} ${message.trim()}`;
      return;
    }
    const result = await response.json();
    const params = currentParams();
    params.set("passage", result.passageId);
    params.set("panel", "repair");
    pushParams(params);
    await loadReader();
  });
}

repairUndo.addEventListener("click", async () => {
  if (!readerState) {
    return;
  }
  const response = await fetch("/api/repair/undo", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      userId: readerState.progress.userId,
      opinionId: readerState.opinion.opinionId,
    }),
  });
  if (!response.ok) {
    const message = await response.text();
    repairAnchor.textContent = `Undo failed: ${response.status} ${message.trim()}`;
    return;
  }
  const result = await response.json();
  const params = currentParams();
  params.set("passage", result.passageId);
  params.set("panel", "repair");
  pushParams(params);
  await loadReader();
});

window.addEventListener("popstate", () => {
  loadReader();
});

loadReader();
