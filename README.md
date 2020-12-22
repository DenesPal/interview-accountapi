# About the solution

## Author

https://github.com/DenesPal

This is my first ever program in Go.

## About some technical decisions

### ApiClient

I wanted a somewhat error-resistant client, and considered the REST API idempotent, so I made `ApiClient` to
 automatically retry failed HTTP requests with some restrictions regarding the status codes.
 However this has introduced some challenges.

The request body has to be replayed for successive retries. `ApiClient.Do` reads the request body into a buffer,
 and recreates a reader for each request. (Alternatively this could be done with a seekable reader.)
 
For Create action, if the first request fails in a state when the action was completed but the reply got lost,
 retrying the POST request will result in a Conflict error. This gets handled by `CreateAccount`: first checks whether
 a resource with the same id exists, (crafting a Conflict error if so,) before calling the POST request through the
 auto-retry method, and if that results in a Conflict error, (considering the ) Fetches the resource by id (to return
  the latest version
  as if POST
 would do). Conflict error of the POST request would be raised only if Fetch was not successful (which is weird).
  
### ApiError

`ApiError` type is extended with `StatusCode` property to save the status code of the HTTP response. This comes handy
 when checking for a certain errors, like the non-existance of a resource. (Test of Delete action for example.)

### List action and pagination

`ListAccounts` return results as `AccountListResults` which consists of a `Channel` of results, an `Error` property,
 and a `Close` method. An internal go-routine of `ListAccounts` feeds the `Account` results into the channel, and
 fetches the next page when the last item of the current page gets consumed. It terminates on the first error,
 (therefore the `Error` is not a channel) and exits if signaled on the `closing` channel (buffered 1).

Consumers shall iterate over the results channel, call the `Close` method when finished, then check the `Error`
 property. `Close` will never block. If just closed while fetching the next page, the goroutine could stay busy with the
 HTTP request, but would exit with the first result or when the request times out. This seems a reasonable tradeoff.

### Validation and defaults

For simplicity the validation and defaults are handled by the same method, however they should be separated for
 anything more complete than this. For example, in case of PATCH, we want to validate the values of the fields, but may
  not want ot enforce required fields and defaults if PATCH accepts an incomplete document of changes only.


-----


# Form3 Take Home Exercise

## Instructions
The goal of this exercise is to write a client library in Go to access our fake account API, which is provided as a Docker
container in the file `docker-compose.yaml` of this repository. Please refer to the
[Form3 documentation](http://api-docs.form3.tech/api.html#organisation-accounts) for information on how to interact with the API.

If you encounter any problems running the fake account API we would encourage you to do some debugging first,
before reaching out for help.

### The solution is expected to
- Be written in Go
- Contain documentation of your technical decisions
- Implement the `Create`, `Fetch`, `List` and `Delete` operations on the `accounts` resource. Note that filtering of the List operation is not required, but you should support paging
- Be well tested to the level you would expect in a commercial environment. Make sure your tests are easy to read.

#### Docker-compose
 - Add your solution to the provided docker-compose file
 - We should be able to run `docker-compose up` and see your tests run against the provided account API service 

### Please don't
- Use a code generator to write the client library
- Use (copy or otherwise) code from any third party without attribution to complete the exercise, as this will result in the test being rejected
- Use a library for your client (e.g: go-resty). Only test libraries are allowed.
- Implement an authentication scheme
- Implement support for the fields `data.attributes.private_identification`, `data.attributes.organisation_identification`
  and `data.relationships`, as they are omitted in the provided fake account API implementation
  
## How to submit your exercise
- Include your name in the README. If you are new to Go, please also mention this in the README so that we can consider this when reviewing your exercise
- Create a private [GitHub](https://help.github.com/en/articles/create-a-repo) repository, copy the `docker-compose` from this repository
- [Invite](https://help.github.com/en/articles/inviting-collaborators-to-a-personal-repository) @form3tech-interviewer-1 to your private repo
- Let us know you've completed the exercise using the link provided at the bottom of the email from our recruitment team

## License
Copyright 2019-2020 Form3 Financial Cloud

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
