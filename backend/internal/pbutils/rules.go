package pbutils

import "github.com/pocketbase/pocketbase/core"

func SetRules(app core.App, name string, list, view, create, update, del string) error {
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return err
	}
	collection.ListRule = RulePtr(list)
	collection.ViewRule = RulePtr(view)
	collection.CreateRule = RulePtr(create)
	collection.UpdateRule = RulePtr(update)
	collection.DeleteRule = RulePtr(del)
	return app.Save(collection)
}

func RulePtr(value string) *string {
	if value == "" {
		return nil
	}
	rule := value
	return &rule
}
