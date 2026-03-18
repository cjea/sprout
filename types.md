# MVP Type Surface: Supreme Court Opinion Reader

## Core Job Answers

The top test for this document is simple: each core user job should map to explicit functions, and every function input should come from either:

- user input
- persistent storage
- an earlier function output
- a primitive value

### User imports a Supreme Court opinion PDF.

```hs
enterUrl   :: () -> Url
generateId :: () -> OpinionId
fetchPdf   :: Url -> IO PdfBytes
storePdf   :: OpinionId -> PdfBytes -> Storage -> IO RawPdf
```

This means the URL comes from the user, the bytes come from the network, and the persisted PDF becomes the canonical source artifact.

### System parses the opinion into logical sections.

```hs
parsePdf :: RawPdf -> IO ParsedPdf
extractMeta :: ParsedPdf -> Meta
guessSections :: Model -> ParsedPdf -> IO [Section]
```

This means section structure is derived from the parsed PDF, not invented elsewhere.

### System breaks the opinion into bite-sized passages.

```hs
chunkSections :: ChunkPolicy -> [Section] -> [Passage]
fitPassage :: ScreenPolicy -> Passage -> PassageFit
repairPassages :: ScreenPolicy -> [Passage] -> [Passage]
```

This means passages are a direct output of sections plus chunking policy.

### User reads one passage at a time and resumes later.

```hs
buildQueue :: UserId -> Opinion -> [Passage] -> Queue
nextPassage :: Queue -> Maybe PassageId
openPassage :: UserId -> PassageId -> Storage -> IO ReadingState
completePassage :: UserId -> PassageId -> Storage -> IO Progress
resumeReading :: UserId -> OpinionId -> Storage -> IO ReadingState
```

This means reading state is reconstructed from stored progress and a persisted opinion.

### User asks a question without losing their place.

```hs
selectSpan :: Passage -> Offset -> Offset -> Span
anchorSpan :: OpinionId -> SectionId -> PassageId -> Span -> Anchor
askQuestion :: UserId -> Anchor -> QuestionText -> Clock -> Question
saveQuestion :: Question -> Storage -> IO Question
```

This means a `Question` is grounded in a selected span, and its anchor is explicit.

### System explains the passage in context.

```hs
gatherContext :: Opinion -> Passage -> Anchor -> [Question] -> Context
guessAnswer :: Model -> Context -> Question -> IO AnswerDraft
```

This means answers are generated from source context, not from a free-floating query.

### System shows cited cases and statutes for the current passage.

```hs
extractCitations :: Passage -> [Citation]
normalizeCitations :: [Citation] -> [Citation]
```

This means related materials are derived from the current passage text.

## Domain Concepts

These are the core nouns for the MVP.

- `Opinion`: one Supreme Court opinion in normalized form
- `Section`: one logical section of the opinion, such as syllabus, majority, concurrence, or dissent
- `Passage`: one bite-sized reading unit, defaulting toward one sentence and never more than three
- `Anchor`: a durable pointer into a passage span
- `Question`: a user-authored question rooted at an anchor
- `AnswerDraft`: a generated answer draft grounded in context
- `Citation`: one cited case, statute, constitutional reference, or internal reference
- `Queue`: the ordered reading plan for one user and one opinion
- `ReadingState`: everything needed to render the current reading screen

## Primitive Types

```hs
type Url = String
type UserId = String
type OpinionId = String
type SectionId = String
type PassageId = String
type QuestionId = String
type CitationId = String
type JusticeName = String
type QuestionText = String
type AnswerText = String
type PdfBytes = Bytes
type Text = String
type Timestamp = String
type PageNo = Int
type Offset = Int
type SentenceNo = Int
```

## Input Origins

This section makes provenance explicit.

```hs
type UserInput =
  { urlInput :: Url
  }

type Clock =
  { now :: () -> Timestamp
  }

type Storage =
  { saveRawPdf :: RawPdf -> IO RawPdf
  , saveOpinion :: Opinion -> IO Opinion
  , savePassages :: [Passage] -> IO [Passage]
  , saveProgress :: Progress -> IO Progress
  , saveQuestionRecord :: Question -> IO Question
  , loadRawPdf :: OpinionId -> IO RawPdf
  , loadOpinion :: OpinionId -> IO Opinion
  , loadPassage :: PassageId -> IO Passage
  , loadProgress :: UserId -> OpinionId -> IO Progress
  , loadQuestions :: UserId -> OpinionId -> IO [Question]
  }

type Model =
  { modelName :: String
  , maxContextTokens :: Int
  }
```

## Source Artifact Types

```hs
type RawPdf =
  { opinionId :: OpinionId
  , sourceUrl :: Url
  , bytes :: PdfBytes
  , fetchedAt :: Timestamp
  , sha256 :: String
  }

type ParsedPdf =
  { opinionId :: OpinionId
  , pages :: [ParsedPage]
  , fullText :: Text
  , warnings :: [ParseWarning]
  }

type ParsedPage =
  { pageNo :: PageNo
  , text :: Text
  , blocks :: [TextBlock]
  }

type TextBlock =
  { pageNo :: PageNo
  , startOffset :: Offset
  , endOffset :: Offset
  , text :: Text
  }

type ParseWarning =
  { code :: String
  , message :: String
  }
```

## Opinion Types

```hs
data SectionKind
  = Syllabus
  | Majority
  | Concurrence
  | Dissent
  | Appendix
  | UnknownSection

type Meta =
  { caseName :: String
  , docketNo :: String
  , decidedOn :: String
  , termLabel :: String
  , primaryAuthor :: Maybe JusticeName
  }

type Section =
  { sectionId :: SectionId
  , kind :: SectionKind
  , title :: String
  , author :: Maybe JusticeName
  , startPage :: PageNo
  , endPage :: PageNo
  , text :: Text
  }

type Opinion =
  { opinionId :: OpinionId
  , meta :: Meta
  , sections :: [Section]
  , fullText :: Text
  }
```

## Passage Types

The POC depends on extremely small passages.

```hs
type ChunkPolicy =
  { targetSentences :: Int
  , maxSentences :: Int
  , preferSingleSentence :: Bool
  , keepSectionBoundaries :: Bool
  , keepCitationContext :: Bool
  }

type ScreenPolicy =
  { maxRenderedLines :: Int
  , maxCharacters :: Int
  , requireFullFit :: Bool
  }

type Passage =
  { passageId :: PassageId
  , opinionId :: OpinionId
  , sectionId :: SectionId
  , sentenceRange :: (SentenceNo, SentenceNo)
  , pageRange :: (PageNo, PageNo)
  , text :: Text
  , citations :: [Citation]
  , fitsOnScreen :: Bool
  }

data PassageFit
  = FitsScreen
  | TooLong
  | NeedsRepair
```

## Anchor and Question Types

```hs
type Span =
  { startOffset :: Offset
  , endOffset :: Offset
  , quote :: Text
  }

type Anchor =
  { opinionId :: OpinionId
  , sectionId :: SectionId
  , passageId :: PassageId
  , span :: Span
  }

type Question =
  { questionId :: QuestionId
  , userId :: UserId
  , anchor :: Anchor
  , text :: QuestionText
  , askedAt :: Timestamp
  , status :: QuestionStatus
  }

data QuestionStatus
  = Open
  | Answered
  | Deferred
```

## Answer Types

```hs
type Context =
  { opinion :: Opinion
  , activePassage :: Passage
  , anchor :: Anchor
  , openQuestions :: [Question]
  , nearbyCitations :: [Citation]
  }

type Evidence =
  { anchor :: Anchor
  , quote :: Text
  , label :: String
  }

type AnswerDraft =
  { questionId :: QuestionId
  , answer :: AnswerText
  , evidence :: [Evidence]
  , caveats :: [String]
  , generatedAt :: Timestamp
  , modelName :: String
  }
```

## Citation Types

```hs
data CitationKind
  = CaseCitation
  | StatuteCitation
  | ConstitutionCitation
  | InternalCitation
  | UnknownCitation

type Citation =
  { citationId :: CitationId
  , kind :: CitationKind
  , rawText :: String
  , normalized :: Maybe String
  , span :: Span
  }
```

## Reading Types

```hs
type Queue =
  { userId :: UserId
  , opinionId :: OpinionId
  , pending :: [PassageId]
  }

type Progress =
  { userId :: UserId
  , opinionId :: OpinionId
  , currentPassage :: Maybe PassageId
  , completedPassages :: [PassageId]
  , openQuestionIds :: [QuestionId]
  , updatedAt :: Timestamp
  }

type Trail =
  { originPassage :: PassageId
  , activePassage :: PassageId
  , questionStack :: [QuestionId]
  }

type ReadingState =
  { opinion :: Opinion
  , passage :: Passage
  , citations :: [Citation]
  , progress :: Progress
  , trail :: Trail
  }
```

## MVP Interfaces

These are the complete high-level interfaces for the MVP.

### Identify the target opinion

```hs
enterUrl :: () -> Url
makeOpinionId :: Url -> OpinionId
```

### Ingest the PDF

```hs
fetchPdf :: Url -> IO PdfBytes
makeRawPdf :: OpinionId -> Url -> PdfBytes -> Timestamp -> RawPdf
storePdf :: OpinionId -> PdfBytes -> Storage -> IO RawPdf
```

### Parse and normalize the opinion

```hs
parsePdf :: RawPdf -> IO ParsedPdf
extractMeta :: ParsedPdf -> Meta
guessSections :: Model -> ParsedPdf -> IO [Section]
buildOpinion :: OpinionId -> Meta -> [Section] -> ParsedPdf -> Opinion
storeOpinion :: Opinion -> Storage -> IO Opinion
```

### Produce bite-sized passages

```hs
chunkSections :: ChunkPolicy -> [Section] -> [Passage]
attachCitations :: [Passage] -> [Passage]
fitPassage :: ScreenPolicy -> Passage -> PassageFit
repairPassages :: ScreenPolicy -> [Passage] -> [Passage]
storePassages :: [Passage] -> Storage -> IO [Passage]
```

### Build the reading loop

```hs
buildQueue :: UserId -> Opinion -> [Passage] -> Queue
nextPassage :: Queue -> Maybe PassageId
openPassage :: UserId -> PassageId -> Storage -> IO ReadingState
completePassage :: UserId -> PassageId -> Storage -> IO Progress
resumeReading :: UserId -> OpinionId -> Storage -> IO ReadingState
```

### Ask and answer questions

```hs
selectSpan :: Passage -> Offset -> Offset -> Span
anchorSpan :: OpinionId -> SectionId -> PassageId -> Span -> Anchor
askQuestion :: UserId -> Anchor -> QuestionText -> Clock -> Question
saveQuestion :: Question -> Storage -> IO Question
gatherContext :: Opinion -> Passage -> Anchor -> [Question] -> Context
guessAnswer :: Model -> Context -> Question -> IO AnswerDraft
```

### Extract and show citations

```hs
extractCitations :: Passage -> [Citation]
normalizeCitations :: [Citation] -> [Citation]
```

## Constructors and Derived Values

If a type appears as input, this section should show where it comes from.

```hs
makeOpinionId :: Url -> OpinionId
makeRawPdf :: OpinionId -> Url -> PdfBytes -> Timestamp -> RawPdf
buildOpinion :: OpinionId -> Meta -> [Section] -> ParsedPdf -> Opinion
buildQueue :: UserId -> Opinion -> [Passage] -> Queue
selectSpan :: Passage -> Offset -> Offset -> Span
anchorSpan :: OpinionId -> SectionId -> PassageId -> Span -> Anchor
askQuestion :: UserId -> Anchor -> QuestionText -> Clock -> Question
```

## End-to-End Flow

This is the MVP as a typed chain.

```hs
()
  -> Url
  -> PdfBytes
  -> RawPdf
  -> ParsedPdf
  -> Meta
  -> [Section]
  -> Opinion
  -> [Passage]
  -> Queue
  -> ReadingState
  -> Span
  -> Anchor
  -> Question
  -> AnswerDraft
```

And in pseudocode:

```hs
runMvp :: UserId -> UserInput -> Storage -> Clock -> Model -> IO ReadingState
runMvp user input storage clock model = do
  let url = urlInput input
  let opinionId = makeOpinionId url

  bytes <- fetchPdf url
  let raw = makeRawPdf opinionId url bytes (now clock ())
  _ <- saveRawPdf storage raw

  parsed <- parsePdf raw
  let meta = extractMeta parsed
  sections <- guessSections model parsed
  let opinion = buildOpinion opinionId meta sections parsed
  _ <- saveOpinion storage opinion

  let chunkPolicy =
        { targetSentences = 1
        , maxSentences = 3
        , preferSingleSentence = True
        , keepSectionBoundaries = True
        , keepCitationContext = True
        }

  let screenPolicy =
        { maxRenderedLines = 18
        , maxCharacters = 900
        , requireFullFit = True
        }

  let rawPassages = chunkSections chunkPolicy sections
  let citedPassages = attachCitations rawPassages
  let fittedPassages = repairPassages screenPolicy citedPassages
  _ <- savePassages storage fittedPassages

  let queue = buildQueue user opinion fittedPassages
  case nextPassage queue of
    Nothing -> fail "No passages available"
    Just passageId -> openPassage user passageId storage
```

## Notes On Naming

- `Section` is enough because the only document type is a Supreme Court opinion.
- `Passage` is better than `ReadSlice` because it reads like an actual reading unit.
- `Question` is better than `Curiosity` because it matches the user action directly.
- `Opinion` is better than `Spine` because it names the actual normalized thing being stored and read.
- Prefixing probabilistic functions with `guess` makes LLM boundaries explicit and encourages downstream review or correction.
- `Authority` was removed because it was too abstract for the MVP. `Citation` is the concrete thing we actually extract, normalize, persist, and show.
