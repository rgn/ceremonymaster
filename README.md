# CeremonyMaster

<img src="./Designer.png" align="right"
     alt="CeremonyMaster logo by ChatGPT" width="120" height="178">

Tool to conduct pastry tasting an collect in a structure way the feedback from the reviwers.

<p align="center">
  <img src="./demo.gif" alt="Demo" width="738">
</p>


## How It Works

1. Download the binary
2. Run the binary
3. Should be self explanatory

### Configuration

Once the application was started a configuration file will be created.

```json
{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "$ref": "#/definitions/Welcome1",
    "definitions": {
        "Welcome1": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "datacollection": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/Datacollection"
                    }
                },
                "evaluation": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/Evaluation"
                    }
                },
                "skilllevels": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/Skilllevel"
                    }
                }
            },
            "required": [
                "datacollection",
                "evaluation",
                "skilllevels"
            ],
            "title": "Welcome1"
        },
        "Datacollection": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "key": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "fields": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/DatacollectionField"
                    }
                }
            },
            "required": [
                "description",
                "fields",
                "key",
                "name"
            ],
            "title": "Datacollection"
        },
        "DatacollectionField": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "type": {
                    "type": "string"
                },
                "key": {
                    "type": "string"
                },
                "title": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "mandatory": {
                    "type": "boolean"
                },
                "options": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "affirmative": {
                    "type": "string"
                },
                "negative": {
                    "type": "string"
                },
                "require_yes": {
                    "type": "boolean"
                }
            },
            "required": [
                "description",
                "key",
                "mandatory",
                "title",
                "type"
            ],
            "title": "DatacollectionField"
        },
        "Evaluation": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "key": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "fields": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/EvaluationField"
                    }
                }
            },
            "required": [
                "description",
                "fields",
                "key",
                "name"
            ],
            "title": "Evaluation"
        },
        "EvaluationField": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "type": {
                    "$ref": "#/definitions/Type"
                },
                "key": {
                    "$ref": "#/definitions/Key"
                },
                "title": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "mandatory": {
                    "type": "boolean"
                },
                "weight": {
                    "type": "integer"
                }
            },
            "required": [
                "description",
                "key",
                "mandatory",
                "title",
                "type"
            ],
            "title": "EvaluationField"
        },
        "Skilllevel": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "level": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "min_points": {
                    "type": "number"
                }
            },
            "required": [
                "description",
                "level",
                "min_points",
                "name"
            ],
            "title": "Skilllevel"
        },
        "Key": {
            "type": "string",
            "enum": [
                "rating",
                "comment"
            ],
            "title": "Key"
        },
        "Type": {
            "type": "string",
            "enum": [
                "range",
                "text"
            ],
            "title": "Type"
        }
    }
}
```