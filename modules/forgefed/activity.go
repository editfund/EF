// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"time"

	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// ForgeLike activity data type
// swagger:model
type ForgeLike struct {
	// swagger:ignore
	ap.Activity
}

func NewForgeLike(actorIRI, objectIRI string, startTime time.Time) (ForgeLike, error) {
	result := ForgeLike{}
	result.Type = ap.LikeType
	result.Actor = ap.IRI(actorIRI)   // That's us, a User
	result.Object = ap.IRI(objectIRI) // That's them, a Repository
	result.StartTime = startTime
	if valid, err := validation.IsValid(result); !valid {
		return ForgeLike{}, err
	}
	return result, nil
}

type ForgeUndoLike struct {
	// swagger:ignore
	ap.Activity
}

func NewForgeUndoLike(actorIRI, objectIRI string, startTime time.Time) (ForgeUndoLike, error) {
	result := ForgeUndoLike{}
	result.Type = ap.UndoType
	result.Actor = ap.IRI(actorIRI) // That's us, a User
	result.Object, _ = NewForgeLike(actorIRI, objectIRI, startTime)
	result.StartTime = startTime
	/*if valid, err := validation.IsValid(result); !valid {
		return ForgeUndoLike{}, err
	}*/
	return result, nil
}

func (like ForgeLike) MarshalJSON() ([]byte, error) {
	return like.Activity.MarshalJSON()
}

// func (like ForgeLike) MarshalJSON() ([]byte, error) {
// 	return like.Activity.MarshalJSON()
// }

func (like *ForgeLike) UnmarshalJSON(data []byte) error {
	return like.Activity.UnmarshalJSON(data)
}

func (undo *ForgeUndoLike) UnmarshalJSON(data []byte) error {
	return undo.Activity.UnmarshalJSON(data)
}

func (like ForgeLike) IsNewer(compareTo time.Time) bool {
	return like.StartTime.After(compareTo)
}

func (like ForgeLike) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(like.Type), "type")...)
	result = append(result, validation.ValidateOneOf(string(like.Type), []any{"Like"}, "type")...)
	if like.Actor == nil {
		result = append(result, "Actor should not be nil.")
	} else {
		result = append(result, validation.ValidateNotEmpty(like.Actor.GetID().String(), "actor")...)
	}
	if like.Object == nil {
		result = append(result, "Object should not be nil.")
	} else {
		result = append(result, validation.ValidateNotEmpty(like.Object.GetID().String(), "object")...)
	}
	result = append(result, validation.ValidateNotEmpty(like.StartTime.String(), "startTime")...)
	if like.StartTime.IsZero() {
		result = append(result, "StartTime was invalid.")
	}

	return result
}

func (undo ForgeUndoLike) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(undo.Type), "type")...)
	result = append(result, validation.ValidateOneOf(string(undo.Type), []any{"Undo"}, "type")...)

	if undo.Actor == nil {
		result = append(result, "Actor should not be nil.")
	}

	fmt.Printf("pre ausgabe %v", undo)

	if undo.Object == nil {
		result = append(result, "Object should not be nil.")
	} else {
		result = append(result, validation.ValidateNotEmpty(string(undo.Object.GetType()), "object.type")...)
		result = append(result, validation.ValidateOneOf(string(undo.Object.GetType()), []any{"Like"}, "object.type")...)
	}

	/*
			} else {
			result = append(result, "Object.type should not be empty")
			//result = append(result, validation.ValidateNotEmpty(undo.Object.GetID().String(), "object")...)
			//fmt.Printf("inner ausgabe %v", undo.Object)
		}
		fmt.Printf("post ausgabe %v", undo.Object)
	*/

	result = append(result, validation.ValidateNotEmpty(undo.StartTime.String(), "startTime")...)
	if undo.StartTime.IsZero() {
		result = append(result, "StartTime was invalid.")
	}
	fmt.Printf("result %v\n", result)
	return result
}
