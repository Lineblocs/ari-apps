package resources

// DIDFlowUnconfigured holds the hard-coded JSON payload representing the default 
// call flow for an unconfigured DID (Direct Inward Dialing) number. This acts as 
// a "warning" or placeholder flow until a user-defined flow is set.
// We use a raw string literal (`...`) to avoid escaping double quotes within the JSON.

const DIDFlowUnconfiguredJSON = `{
  "graph": {
    "cells": [
      {
        "type": "devs.FlowLink",
        "source": {
          "id": "63a1a711-9e56-441e-9388-f4bbc6d2b400",
          "port": "Incoming Call"
        },
        "target": {
          "id": "ce811673-46c8-4bb8-9eaf-a28152c90bde",
          "port": "In"
        },
        "id": "8faf6ddb-afee-40ce-9e8a-c6bde9872611",
        "z": 0,
        "vertices": [
          {
            "x": 609.0025,
            "y": 234.861
          }
        ],
        "attrs": {}
      },
      {
        "name": "Launch",
        "type": "devs.LaunchModel",
        "size": {
          "width": 320,
          "height": 100
        },
        "inPorts": [],
        "outPorts": [
          "Incoming Call"
        ],
        "ports": {
          "groups": {
            "in": {
              "position": "top",
              "label": {
                "position": {
                  "name": "manual",
                  "args": {
                    "y": -10,
                    "x": -140,
                    "attrs": {
                      ".": {
                        "text-anchor": "middle"
                      }
                    }
                  }
                }
              },
              "attrs": {
                ".port-label": {
                  "ref-x": -140,
                  "fill": "#385374"
                },
                ".port-body": {
                  "r": 5,
                  "ref-x": -140,
                  "ref-y": 0,
                  "stroke-width": 2,
                  "stroke": "#36D576",
                  "fill": "#36D576",
                  "padding": 20,
                  "transform": "matrix(1 0 0 1 0 2)",
                  "magnet": true
                }
              }
            },
            "out": {
              "position": "bottom",
              "label": {
                "position": "outside"
              },
              "attrs": {
                ".port-label": {
                  "fill": "#385374"
                },
                ".port-body": {
                  "r": 5,
                  "ref-x": 0,
                  "ref-y": 0,
                  "stroke-width": 5,
                  "stroke": "#385374",
                  "fill": "#000878",
                  "padding": 2,
                  "transform": "matrix(1 0 0 1 0 2)",
                  "magnet": true
                }
              }
            }
          },
          "items": [
            {
              "id": "Incoming Call",
              "group": "out",
              "attrs": {
                ".port-label": {
                  "text": "Incoming Call"
                }
              }
            }
          ]
        },
        "position": {
          "x": 448.005,
          "y": 84.72199999999998
        },
        "angle": 0,
        "id": "63a1a711-9e56-441e-9388-f4bbc6d2b400",
        "z": 1,
        "attrs": {
          ".label": {
            "ref-y": 20,
            "font-size": "18",
            "fill": "#385374"
          }
        }
      },
      {
        "name": "Playback",
        "type": "devs.PlaybackModel",
        "size": {
          "width": 320,
          "height": 100
        },
        "inPorts": [
          "In"
        ],
        "outPorts": [
          "Finished"
        ],
        "ports": {
          "groups": {
            "in": {
              "position": "top",
              "label": {
                "position": {
                  "name": "manual",
                  "args": {
                    "y": -10,
                    "x": -140,
                    "attrs": {
                      ".": {
                        "text-anchor": "middle"
                      }
                    }
                  }
                }
              },
              "attrs": {
                ".port-label": {
                  "ref-x": -140,
                  "fill": "#385374"
                },
                ".port-body": {
                  "r": 5,
                  "ref-x": -140,
                  "ref-y": 0,
                  "stroke-width": 2,
                  "stroke": "#36D576",
                  "fill": "#36D576",
                  "padding": 20,
                  "transform": "matrix(1 0 0 1 0 2)",
                  "magnet": true
                }
              }
            },
            "out": {
              "position": "bottom",
              "label": {
                "position": "outside"
              },
              "attrs": {
                ".port-label": {
                  "fill": "#385374"
                },
                ".port-body": {
                  "r": 5,
                  "ref-x": 0,
                  "ref-y": 0,
                  "stroke-width": 5,
                  "stroke": "#385374",
                  "fill": "#000878",
                  "padding": 2,
                  "transform": "matrix(1 0 0 1 0 2)",
                  "magnet": true
                }
              }
            }
          },
          "items": [
            {
              "id": "In",
              "group": "in",
              "attrs": {
                ".port-label": {
                  "text": "In"
                }
              }
            },
            {
              "id": "Finished",
              "group": "out",
              "attrs": {
                ".port-label": {
                  "text": "Finished"
                }
              }
            }
          ]
        },
        "position": {
          "x": 450,
          "y": 285
        },
        "angle": 0,
        "id": "ce811673-46c8-4bb8-9eaf-a28152c90bde",
        "z": 2,
        "attrs": {
          ".label": {
            "fill": "#385374",
            "font-size": "18"
          },
          ".description": {
            "text": "Use text to speech to create playback",
            "fill": "#385374"
          }
        }
      }
    ]
  },
  "models": [
    {
      "id": "63a1a711-9e56-441e-9388-f4bbc6d2b400",
      "name": "Launch",
      "data": {},
      "links": []
    },
    {
      "id": "ce811673-46c8-4bb8-9eaf-a28152c90bde",
      "name": "Playback",
      "data": {
        "playback_type": "Say",
        "text_to_say": "This number hasn't been configured yet.",
        "text_language": "en-US",
        "text_gender": "FEMALE",
        "voice": "en-US-Standard-C",
        "number_of_loops": "1"
      },
      "links": []
    }
  ]
}`