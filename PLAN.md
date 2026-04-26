We are developing a personal news digest website, self-hosted.

## features

- multi-user
- a website where news articles are organized via AI an presented interestingly
- sources:
  - a user can add a FreshRSS instance (url, auth) so the news digets pulls theirs news off there
  - a user can also add separate rss feeds manually
- a user can define their interests in multiple free-text fields (think of a "wizard" at the start). of ourse this wizard can be customized thrugh the user profile later
- the news should be daily auto generated
  - the generator has a default prompt, then pulls all news articles from the feeds in a large prompt and submits them to an AI (pull a standard library so AI models and endpoints can be configured (use env vars))
  - the user should be able to manually re-generate the news digest
  - after the news are generated, save them to a database. the generation should only happpen once daily or on demand
  - the user should be able to up/downvote articles to improve the algorith, make not of that in the prompt
  - if the prompt is too large, logically split it up into multiple prompts
  - the news should be ordered by relevance/interest, also add a mechanism for that
  - add a defualt set of categories (tech, health, economy and so on) and correctly categorize them
  - news can be multi-language. represent the "summarized article" in the original language
  - of course, think of a way that the response from the AI model is transformed into structured data to store into the database
  - the user shuld also be able to add a custom section to the news digest (for example, i want all important cyber security breaches of the last 24h in one place). think of a good way to implement that, the user should be able to confugure that in natural language
- the user should be able to jump back day by day to previous news digests
  - if a user generates a manual digest, also save it and dont override the automatical digest for the day
- add a configurable database (sqlite by defualt, also postgres)
- no javascript at all. only use native html elements, eg dialogs etc. do not use javascript, onclick="" etc.
- when showing remote images: add a proxy endpoint that proxys images through a route
- add a .env file and .env.example

## configurable

- add env var to enable/disable user registrations

## self-hsoting

- dockerfile with minimal setup
- docker compose example

## code style

- only use Golang
- comments only where strictly necessary
- choose fitting go templating engine

## styling

- use tailwindcss v4
- see fodler "design-template" for the Claude Design preset. use the "editorial" template. implement it in the exact style/way
- as I've repeated before. DO NOT use javascrpit. implement in Go template you see fit
- use css classes with tailwind for repeated elements. do not copy large sets of classes 1 to 1
- implement darkmode (device scheme)

## git

- commit your changes automatically after each logical step
- DO NOT add co-authored-by. NEVER
- do not add commit descriptions
- use short descriptive commit messages
- do not push

## additions

- authentication: through email/password, do not send mail verifications
- use established ways of atuehtnications in go/go libraries, dont roll it on your own
- make features abstract (eg. sources) so they can be enhacned later
