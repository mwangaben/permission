package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RoleHasPermission struct{ ent.Schema }

func (RoleHasPermission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "role_has_permissions"},
	}
}

func (RoleHasPermission) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id").Unique().Immutable(),
		field.Uint("permission_id"),
		field.Uint("role_id"),
		field.String("tenant_id").MaxLen(100).Optional().Nillable(),
	}
}

func (RoleHasPermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("permission_id", "role_id").Unique().
			StorageKey("idx_role_permissions_unique"),
		index.Fields("tenant_id").StorageKey("idx_role_permissions_tenant"),
	}
}
